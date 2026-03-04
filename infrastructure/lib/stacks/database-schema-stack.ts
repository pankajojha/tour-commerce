import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as secretsmanager from 'aws-cdk-lib/aws-secretsmanager';
import * as cr from 'aws-cdk-lib/custom-resources';
import * as logs from 'aws-cdk-lib/aws-logs';
import { Construct } from 'constructs';
import * as path from 'path';

export interface DatabaseSchemaStackProps extends cdk.StackProps {
  vpc: ec2.Vpc;
  lambdaSecurityGroup: ec2.SecurityGroup;
  databaseSecurityGroup: ec2.SecurityGroup;
  databaseCredentialsSecret: secretsmanager.Secret;
  databaseEndpoint: string;
}

export class DatabaseSchemaStack extends cdk.Stack {
  public readonly schemaDeployerFunction: lambda.Function;

  constructor(scope: Construct, id: string, props: DatabaseSchemaStackProps) {
    super(scope, id, props);

    // Create log group for Lambda
    const schemaDeployerLogGroup = new logs.LogGroup(this, 'SchemaDeployerLogGroup', {
      retention: logs.RetentionDays.ONE_WEEK,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // Lambda function for schema deployment
    this.schemaDeployerFunction = new lambda.Function(this, 'SchemaDeployerFunction', {
      runtime: lambda.Runtime.NODEJS_18_X,
      handler: 'index.handler',
      code: lambda.Code.fromAsset(path.join(__dirname, '../../database'), {
        bundling: {
          image: lambda.Runtime.NODEJS_18_X.bundlingImage,
          command: [
            'bash', '-c',
            [
              'cd schema-deployer-lambda',
              'npm ci',
              'npm run build',
              'cp -r dist/* /asset-output/',
              'cp -r node_modules /asset-output/',
              'cd ..',
              'mkdir -p /asset-output/migrations',
              'cp -r migrations/* /asset-output/migrations/',
            ].join(' && '),
          ],
          local: {
            tryBundle(outputDir: string): boolean {
              try {
                require('child_process').execSync(
                  [
                    'cd schema-deployer-lambda',
                    'npm ci',
                    'npm run build',
                    `cp -r dist/* ${outputDir}/`,
                    `cp -r node_modules ${outputDir}/`,
                    'cd ..',
                    `mkdir -p ${outputDir}/migrations`,
                    `cp -r migrations/* ${outputDir}/migrations/`,
                  ].join(' && '),
                  { cwd: path.join(__dirname, '../../database'), stdio: 'inherit', shell: true },
                );
                return true;
              } catch (e) {
                console.error('Local bundling failed, falling back to Docker:', e);
                return false;
              }
            },
          },
        },
      }),
      timeout: cdk.Duration.minutes(15),
      memorySize: 512,
      vpc: props.vpc,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS,
      },
      securityGroups: [props.lambdaSecurityGroup],
      environment: {
        DB_SECRET_ARN: props.databaseCredentialsSecret.secretArn,
        MIGRATIONS_DIR: '/var/task/migrations',
      },
      logGroup: schemaDeployerLogGroup,
    });

    // Grant Lambda permission to read database credentials
    props.databaseCredentialsSecret.grantRead(this.schemaDeployerFunction);

    // Create log group for custom resource provider
    const providerLogGroup = new logs.LogGroup(this, 'SchemaDeployerProviderLogGroup', {
      retention: logs.RetentionDays.ONE_WEEK,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // Custom resource provider
    const provider = new cr.Provider(this, 'SchemaDeployerProvider', {
      onEventHandler: this.schemaDeployerFunction,
      logGroup: providerLogGroup,
    });

    // Custom resource to trigger schema deployment
    const schemaDeployment = new cdk.CustomResource(this, 'SchemaDeployment', {
      serviceToken: provider.serviceToken,
      properties: {
        // Change this value to trigger redeployment
        Version: '1.0.0',
        Timestamp: Date.now(),
      },
    });

    // Outputs
    new cdk.CfnOutput(this, 'SchemaDeployerFunctionArn', {
      value: this.schemaDeployerFunction.functionArn,
      description: 'ARN of the schema deployer Lambda function',
      exportName: 'ExplorerAgentSchemaDeployerFunctionArn',
    });

    new cdk.CfnOutput(this, 'SchemaDeploymentStatus', {
      value: 'Deployed',
      description: 'Database schema deployment status',
    });
  }
}
