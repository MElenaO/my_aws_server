# AWS BACKEND
A small backend that will return a greeting written in Golang.

Docker Container uses golang image as builder image, and alpine latest as base image

CDK project
- Create VPC and with 3 public subnets
- Create an Application Load Balancer, listening in port 80
- Download container image from ECR
- Create a ECS Cluster
- Create a Fargate Task Definition to run container image, which listens in port 8080
- Create a Fargate Service to deploy the task

Github actions
- build the app
- authenticate to AWS