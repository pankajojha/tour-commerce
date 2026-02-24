import * as cdk from 'aws-cdk-lib';
import * as logs from 'aws-cdk-lib/aws-logs';
import { Construct } from 'constructs';

export class MonitoringStack extends cdk.Stack {
  public readonly apiLogGroup: logs.LogGroup;
  public readonly lambdaLogGroup: logs.LogGroup;
  public readonly databaseLogGroup: logs.LogGroup;
  public readonly redisLogGroup: logs.LogGroup;

  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // CloudWatch Log Group for API Gateway
    this.apiLogGroup = new logs.LogGroup(this, 'ApiGatewayLogGroup', {
      logGroupName: '/aws/apigateway/tour-agent',
      retention: logs.RetentionDays.ONE_MONTH,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // CloudWatch Log Group for Lambda functions
    this.lambdaLogGroup = new logs.LogGroup(this, 'LambdaLogGroup', {
      logGroupName: '/aws/lambda/tour-agent',
      retention: logs.RetentionDays.ONE_MONTH,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // CloudWatch Log Group for Aurora PostgreSQL
    this.databaseLogGroup = new logs.LogGroup(this, 'DatabaseLogGroup', {
      logGroupName: '/aws/rds/cluster/tour-agent-database/postgresql',
      retention: logs.RetentionDays.ONE_MONTH,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // CloudWatch Log Group for ElastiCache Redis
    this.redisLogGroup = new logs.LogGroup(this, 'RedisLogGroup', {
      logGroupName: '/aws/elasticache/tour-agent-redis',
      retention: logs.RetentionDays.ONE_MONTH,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // CloudWatch Log Group for X-Ray traces
    const xrayLogGroup = new logs.LogGroup(this, 'XRayLogGroup', {
      logGroupName: '/aws/xray/tour-agent',
      retention: logs.RetentionDays.ONE_WEEK,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // CloudWatch Log Group for EventBridge
    const eventBridgeLogGroup = new logs.LogGroup(this, 'EventBridgeLogGroup', {
      logGroupName: '/aws/events/tour-agent',
      retention: logs.RetentionDays.ONE_MONTH,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // CloudWatch Log Group for SQS
    const sqsLogGroup = new logs.LogGroup(this, 'SqsLogGroup', {
      logGroupName: '/aws/sqs/tour-agent',
      retention: logs.RetentionDays.ONE_MONTH,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // Outputs
    new cdk.CfnOutput(this, 'ApiLogGroupName', {
      value: this.apiLogGroup.logGroupName,
      description: 'API Gateway log group name',
      exportName: 'TourAgentApiLogGroupName',
    });

    new cdk.CfnOutput(this, 'LambdaLogGroupName', {
      value: this.lambdaLogGroup.logGroupName,
      description: 'Lambda log group name',
      exportName: 'TourAgentLambdaLogGroupName',
    });

    new cdk.CfnOutput(this, 'DatabaseLogGroupName', {
      value: this.databaseLogGroup.logGroupName,
      description: 'Database log group name',
      exportName: 'TourAgentDatabaseLogGroupName',
    });

    new cdk.CfnOutput(this, 'RedisLogGroupName', {
      value: this.redisLogGroup.logGroupName,
      description: 'Redis log group name',
      exportName: 'TourAgentRedisLogGroupName',
    });

    new cdk.CfnOutput(this, 'XRayLogGroupName', {
      value: xrayLogGroup.logGroupName,
      description: 'X-Ray log group name',
      exportName: 'TourAgentXRayLogGroupName',
    });
  }
}
