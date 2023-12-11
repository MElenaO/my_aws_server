package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	dynamodb "github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	ecr "github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	ecs "github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	elb "github.com/aws/aws-cdk-go/awscdk/v2/awselasticloadbalancingv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type InfraStackProps struct {
	awscdk.StackProps
}

func NewInfraStack(scope constructs.Construct, id string, props *InfraStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// Create a VPC
	vpc := awsec2.NewVpc(stack, jsii.String("BaseVpc"),
		&awsec2.VpcProps{
			IpAddresses:       awsec2.IpAddresses_Cidr(jsii.String("10.10.0.0/16")),
			NatGateways:       jsii.Number(0),
			AvailabilityZones: &[]*string{jsii.String("eu-north-1a"), jsii.String("eu-north-1b"), jsii.String("eu-north-1c")},
			// 'subnetConfiguration' specifies the "subnet groups" to create.
			// Every subnet group will have a subnet for each AZ,
			SubnetConfiguration: &[]*awsec2.SubnetConfiguration{
				{
					Name: jsii.String("Application"),
					// 'subnetType' controls Internet access, as described above.
					SubnetType: awsec2.SubnetType_PUBLIC,
				},
			},
		})

	// Create the load balancer in a VPC. 'internetFacing' is 'false'
	// by default, which creates an internal load balancer.
	alb := elb.NewApplicationLoadBalancer(stack, jsii.String("LB"), &elb.ApplicationLoadBalancerProps{
		Vpc:            vpc,
		InternetFacing: jsii.Bool(true),
	})

	// Add a listener and open up the load balancer's security group
	// to the world.
	listener := alb.AddListener(jsii.String("Listener"), &elb.BaseApplicationListenerProps{
		Port: jsii.Number(80),

		// 'open: true' is the default, you can leave it out if you want. Set it
		// to 'false' and use `listener.connections` if you want to be selective
		// about who can access the load balancer.
		Open: jsii.Bool(true),
	})

	// Create DynamoDB table
	table := dynamodb.NewTable(stack, jsii.String("DynamoDBTable"), &dynamodb.TableProps{
		TableName: jsii.String("GreetingTable"),
		PartitionKey: &dynamodb.Attribute{
			Name: jsii.String("id"),
			Type: dynamodb.AttributeType_NUMBER,
		},
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
		//PointInTimeRecovery: jsii.Bool(true),
	})

	// ECS Config
	// Create an ECS cluster
	cluster := ecs.NewCluster(stack, jsii.String("Cluster"), &ecs.ClusterProps{
		Vpc: vpc,
	})

	// Create a reference to the registry.
	// Repository name just needs webserver (rest of URI is added automatically)
	repository := ecr.Repository_FromRepositoryName(stack, jsii.String("WebServerRepo"), jsii.String("webserver"))

	// Define the task
	taskDefinition := ecs.NewFargateTaskDefinition(stack, jsii.String("WebServerTask"), &ecs.FargateTaskDefinitionProps{
		RuntimePlatform: &ecs.RuntimePlatform{
			OperatingSystemFamily: ecs.OperatingSystemFamily_LINUX(),
			CpuArchitecture:       ecs.CpuArchitecture_X86_64(),
		},
		MemoryLimitMiB: jsii.Number(512),
		Cpu:            jsii.Number(256),
	})
	imageTag := awscdk.NewCfnParameter(stack, jsii.String("ImageTag"), &awscdk.CfnParameterProps{
		Type:    jsii.String("String"),
		Default: jsii.String("latest"),
	})

	logGroup := awslogs.NewLogGroup(stack, jsii.String("LogGroup"), &awslogs.LogGroupProps{
		LogGroupName: jsii.String("WebServerLogGroup"),
		Retention:    awslogs.RetentionDays_FIVE_DAYS,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	// Get the image from the registry
	defaultContainer := taskDefinition.AddContainer(jsii.String("DefaultContainer"), &ecs.ContainerDefinitionOptions{
		Image: ecs.ContainerImage_FromEcrRepository(repository, imageTag.ValueAsString()),
		Logging: ecs.NewAwsLogDriver(&ecs.AwsLogDriverProps{
			StreamPrefix: jsii.String("app"),
			LogGroup:     logGroup,
			Mode:         ecs.AwsLogDriverMode_NON_BLOCKING,
		}),
	})

	// Add a port mapping for the image
	defaultContainer.AddPortMappings(&ecs.PortMapping{
		ContainerPort: jsii.Number(8080),
		Protocol:      ecs.Protocol_TCP,
	})

	defaultContainer.AddEnvironment(jsii.String("SERVER_TABLE_NAME"), jsii.String(*table.TableName()))

	// Allow service to read and write from the table
	table.GrantReadWriteData(taskDefinition.TaskRole())

	// Create the Fargate service
	// 2 replicas
	// Public IP needed in order for the Service to access ECR (since no NAT used)
	svc := ecs.NewFargateService(stack, jsii.String("Service"), &ecs.FargateServiceProps{
		Cluster:        cluster,
		TaskDefinition: taskDefinition,
		DesiredCount:   jsii.Number(2),
		AssignPublicIp: jsii.Bool(true),
	})

	// Define the connections between the ALB and the Container, two sec groups are created
	// ALB: Inbound all traffic from port 80, outbound to Container in port 8080
	// Container: Inbound from ALB in port 8080, Outbound all traffic anywhere
	targetGroup := listener.AddTargets(jsii.String("ServiceTarget"), &elb.AddApplicationTargetsProps{
		Port: jsii.Number(80),
		Targets: &[]elb.IApplicationLoadBalancerTarget{
			svc.LoadBalancerTarget(&ecs.LoadBalancerTargetOptions{
				ContainerName: jsii.String("DefaultContainer"),
				ContainerPort: jsii.Number(8080),
			}),
		},
		HealthCheck: &elb.HealthCheck{
			Path: jsii.String("/ping"),
		},
	})
	awscdk.NewCfnOutput(stack, jsii.String("targetGroup"), &awscdk.CfnOutputProps{Value: targetGroup.ToString()})

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewInfraStack(app, "InfraStack", &InfraStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	// 	Account: jsii.String(<account_nr>),
	// 	Region:  jsii.String(<region_id>),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}
