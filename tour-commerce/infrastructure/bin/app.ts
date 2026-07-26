#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { CommerceDatabaseStack } from '../lib/stacks/commerce-database-stack';
import { CommerceApiStack } from '../lib/stacks/commerce-api-stack';
import { CommerceEventsStack } from '../lib/stacks/commerce-events-stack';

const app = new cdk.App();

const env = {
  account: process.env.CDK_DEFAULT_ACCOUNT,
  region:  process.env.CDK_DEFAULT_REGION || 'us-east-1',
};

// ── Import shared VPC and security groups from tour-agent network stack ────────
// These cross-stack references let commerce Lambdas join the same VPC as tour-agent.
const vpcId             = process.env.SHARED_VPC_ID            || app.node.tryGetContext('sharedVpcId');
const lambdaSgId        = process.env.LAMBDA_SG_ID             || app.node.tryGetContext('lambdaSecurityGroupId');
const dbSgId            = process.env.DB_SG_ID                 || app.node.tryGetContext('dbSecurityGroupId');
const cognitoPoolArn    = process.env.COGNITO_USER_POOL_ARN    || app.node.tryGetContext('cognitoUserPoolArn') || '';
const tourAgentApiUrl   = process.env.TOUR_AGENT_API_URL       || app.node.tryGetContext('tourAgentApiUrl')   || '';
const tourPlannerAiUrl  = process.env.TOUR_PLANNER_AI_URL      || app.node.tryGetContext('tourPlannerAiUrl')  || '';

// ── Database stack ─────────────────────────────────────────────────────────────
// Provisions its own Aurora Serverless v2 cluster in the shared VPC so
// commerce data is isolated from the tour-agent database.
import * as ec2 from 'aws-cdk-lib/aws-ec2';

// Reference existing VPC created by tour-agent network stack.
const sharedVpc = ec2.Vpc.fromLookup(app, 'SharedVpc', { vpcId });
const lambdaSg  = ec2.SecurityGroup.fromSecurityGroupId(app, 'LambdaSg', lambdaSgId);
const dbSg      = ec2.SecurityGroup.fromSecurityGroupId(app, 'DbSg', dbSgId);

const dbStack = new CommerceDatabaseStack(app, 'TourCommerceDatabaseStack', {
  env,
  description: 'Aurora Serverless v2 PostgreSQL for tour-commerce',
  vpc:                  sharedVpc,
  lambdaSecurityGroup:  lambdaSg,
  databaseSecurityGroup: dbSg,
});

// ── API stack ──────────────────────────────────────────────────────────────────
const apiStack = new CommerceApiStack(app, 'TourCommerceApiStack', {
  env,
  description: 'Lambda + API Gateway for tour-commerce services',
  vpc:                 sharedVpc,
  lambdaSecurityGroup: lambdaSg,
  dbCredentialsSecret: dbStack.credentialsSecret,
  dbEndpoint:          dbStack.cluster.clusterEndpoint.hostname,
  dbName:              'tourcommerce',
  cognitoUserPoolArn:  cognitoPoolArn,
  tourAgentApiUrl,
  tourPlannerAiUrl,
});
apiStack.addDependency(dbStack);

// ── Events stack ───────────────────────────────────────────────────────────────
const eventsStack = new CommerceEventsStack(app, 'TourCommerceEventsStack', {
  env,
  description: 'EventBridge + SNS + SQS event fabric for tour-commerce',
  ordersFunctionArn:   `arn:aws:lambda:${env.region}:${env.account}:function:commerce-orders`,
  paymentsFunctionArn: `arn:aws:lambda:${env.region}:${env.account}:function:commerce-payments`,
});
eventsStack.addDependency(apiStack);

// ── Tags ───────────────────────────────────────────────────────────────────────
cdk.Tags.of(app).add('Project', 'TourCommerce');
cdk.Tags.of(app).add('ManagedBy', 'CDK');
