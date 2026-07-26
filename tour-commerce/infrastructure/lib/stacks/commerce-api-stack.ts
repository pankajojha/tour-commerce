import * as cdk from 'aws-cdk-lib';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import * as secretsmanager from 'aws-cdk-lib/aws-secretsmanager';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as path from 'path';
import { Construct } from 'constructs';

export interface CommerceApiStackProps extends cdk.StackProps {
  vpc: ec2.IVpc;
  lambdaSecurityGroup: ec2.ISecurityGroup;
  dbCredentialsSecret: secretsmanager.ISecret;
  dbEndpoint: string;
  dbName: string;
  /** ARN of the Cognito User Pool authorizer shared with tour-agent. */
  cognitoUserPoolArn: string;
  /** Base URL of the tour-agent API Gateway for tour-agent client calls. */
  tourAgentApiUrl: string;
  /** Base URL of the tour-planner-ai service (BedrockAgentCore endpoint). */
  tourPlannerAiUrl: string;
}

export class CommerceApiStack extends cdk.Stack {
  public readonly api: apigateway.RestApi;

  constructor(scope: Construct, id: string, props: CommerceApiStackProps) {
    super(scope, id, props);

    // ── shared Lambda environment ─────────────────────────────────────────────
    const commonEnv: Record<string, string> = {
      DB_SECRET_ARN:        props.dbCredentialsSecret.secretArn,
      DB_ENDPOINT:          props.dbEndpoint,
      DB_NAME:              props.dbName,
      TOUR_AGENT_API_URL:   props.tourAgentApiUrl,
      TOUR_PLANNER_AI_URL:  props.tourPlannerAiUrl,
      // DATABASE_URL is resolved at runtime by reading the secret
    };

    const lambdaDefaults: Partial<lambda.FunctionProps> = {
      runtime:       lambda.Runtime.PROVIDED_AL2023,
      handler:       'bootstrap',
      vpc:           props.vpc,
      vpcSubnets:    { subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS },
      securityGroups: [props.lambdaSecurityGroup],
      timeout:       cdk.Duration.seconds(30),
      memorySize:    256,
      environment:   commonEnv,
      logRetention:  logs.RetentionDays.TWO_WEEKS,
    };

    // ── Lambda functions ──────────────────────────────────────────────────────

    const cartFn = new lambda.Function(this, 'CommerceCartFn', {
      ...lambdaDefaults,
      functionName: 'commerce-cart',
      description:  'Cart CRUD operations',
      code: lambda.Code.fromAsset(path.join(__dirname, '../../../services/cart'), {
        bundling: goBundle(),
      }),
    });

    const checkoutFn = new lambda.Function(this, 'CommerceCheckoutFn', {
      ...lambdaDefaults,
      functionName: 'commerce-checkout',
      description:  'Checkout orchestration with tour-agent booking + tour-planner-ai validation',
      code: lambda.Code.fromAsset(path.join(__dirname, '../../../services/checkout'), {
        bundling: goBundle(),
      }),
      environment: {
        ...commonEnv,
        STRIPE_SECRET_KEY: resolveStripeKey(this),
      },
    });

    const ordersFn = new lambda.Function(this, 'CommerceOrdersFn', {
      ...lambdaDefaults,
      functionName: 'commerce-orders',
      description:  'Order lifecycle management',
      code: lambda.Code.fromAsset(path.join(__dirname, '../../../services/orders'), {
        bundling: goBundle(),
      }),
    });

    const paymentsFn = new lambda.Function(this, 'CommercePaymentsFn', {
      ...lambdaDefaults,
      functionName: 'commerce-payments',
      description:  'Stripe payment processing and webhook handling',
      code: lambda.Code.fromAsset(path.join(__dirname, '../../../services/payments'), {
        bundling: goBundle(),
      }),
      environment: {
        ...commonEnv,
        STRIPE_SECRET_KEY:    resolveStripeKey(this),
        STRIPE_WEBHOOK_SECRET: resolveWebhookSecret(this),
      },
    });

    const aiFn = new lambda.Function(this, 'CommerceAiFn', {
      ...lambdaDefaults,
      functionName: 'commerce-ai',
      description:  'AI recommendations + budget check bridge to tour-planner-ai',
      code: lambda.Code.fromAsset(path.join(__dirname, '../../../services/ai-commerce'), {
        bundling: goBundle(),
      }),
    });

    // Grant Secrets Manager access to all functions.
    for (const fn of [cartFn, checkoutFn, ordersFn, paymentsFn, aiFn]) {
      props.dbCredentialsSecret.grantRead(fn);
      fn.addToRolePolicy(new iam.PolicyStatement({
        actions: ['secretsmanager:GetSecretValue'],
        resources: [props.dbCredentialsSecret.secretArn],
      }));
    }

    // ── API Gateway ───────────────────────────────────────────────────────────
    this.api = new apigateway.RestApi(this, 'CommerceApi', {
      restApiName: 'tour-commerce-api',
      description: 'Tour Commerce REST API',
      deployOptions: {
        stageName:       'v1',
        loggingLevel:    apigateway.MethodLoggingLevel.INFO,
        dataTraceEnabled: true,
        metricsEnabled:  true,
        tracingEnabled:  true,
      },
      defaultCorsPreflightOptions: {
        allowOrigins: apigateway.Cors.ALL_ORIGINS,
        allowMethods: apigateway.Cors.ALL_METHODS,
        allowHeaders: ['Content-Type', 'Authorization', 'X-Amz-Date', 'X-Api-Key'],
      },
    });

    const cognitoAuthorizer = new apigateway.CognitoUserPoolsAuthorizer(this, 'CommerceAuthorizer', {
      cognitoUserPools: [
        // Reuse the existing Cognito user pool from tour-agent.
        // We reference it by ARN since it lives in a separate stack.
        apigateway.UserPool.fromUserPoolArn(this, 'ExistingUserPool', props.cognitoUserPoolArn) as any,
      ],
      authorizerName: 'CommerceJWTAuthorizer',
      identitySource: 'method.request.header.Authorization',
    });

    const authOptions: apigateway.MethodOptions = {
      authorizer:       cognitoAuthorizer,
      authorizationType: apigateway.AuthorizationType.COGNITO,
    };

    // ── Route registration ────────────────────────────────────────────────────
    const commerce = this.api.root.addResource('commerce');

    // /commerce/cart
    const cart = commerce.addResource('cart');
    cart.addMethod('POST', new apigateway.LambdaIntegration(cartFn), authOptions);
    const cartActive = cart.addResource('active');
    cartActive.addMethod('GET', new apigateway.LambdaIntegration(cartFn), authOptions);
    const cartId = cart.addResource('{id}');
    cartId.addMethod('GET', new apigateway.LambdaIntegration(cartFn), authOptions);
    const cartItems = cartId.addResource('items');
    cartItems.addMethod('POST', new apigateway.LambdaIntegration(cartFn), authOptions);
    const cartItemId = cartItems.addResource('{item_id}');
    cartItemId.addMethod('PUT', new apigateway.LambdaIntegration(cartFn), authOptions);
    cartItemId.addMethod('DELETE', new apigateway.LambdaIntegration(cartFn), authOptions);

    // /commerce/checkout
    const checkout = commerce.addResource('checkout');
    checkout.addMethod('POST', new apigateway.LambdaIntegration(checkoutFn), authOptions);
    const checkoutSession = checkout.addResource('{session_id}');
    checkoutSession.addMethod('GET', new apigateway.LambdaIntegration(checkoutFn), authOptions);
    checkoutSession.addResource('confirm').addMethod('POST', new apigateway.LambdaIntegration(checkoutFn), authOptions);

    // /commerce/orders
    const orders = commerce.addResource('orders');
    orders.addMethod('GET', new apigateway.LambdaIntegration(ordersFn), authOptions);
    const orderId = orders.addResource('{id}');
    orderId.addMethod('GET', new apigateway.LambdaIntegration(ordersFn), authOptions);
    orderId.addResource('cancel').addMethod('POST', new apigateway.LambdaIntegration(ordersFn), authOptions);

    // /commerce/payments
    const payments = commerce.addResource('payments');
    payments.addResource('intent').addMethod('POST', new apigateway.LambdaIntegration(paymentsFn), authOptions);
    // Stripe webhook is unauthenticated (signature-verified internally).
    payments.addResource('webhook').addMethod('POST', new apigateway.LambdaIntegration(paymentsFn));
    payments.addResource('{order_id}').addResource('refund').addMethod('POST', new apigateway.LambdaIntegration(paymentsFn), authOptions);

    // /commerce/ai
    const ai = commerce.addResource('ai');
    ai.addResource('recommendations').addMethod('POST', new apigateway.LambdaIntegration(aiFn), authOptions);
    ai.addResource('budget-check').addMethod('POST', new apigateway.LambdaIntegration(aiFn), authOptions);
    ai.addResource('upsell').addMethod('POST', new apigateway.LambdaIntegration(aiFn), authOptions);

    // ── outputs ───────────────────────────────────────────────────────────────
    new cdk.CfnOutput(this, 'CommerceApiUrl', {
      value:      this.api.url,
      exportName: 'TourCommerceApiUrl',
    });
  }
}

// ─── helpers ──────────────────────────────────────────────────────────────────

function goBundle(): cdk.BundlingOptions {
  return {
    image: lambda.Runtime.PROVIDED_AL2023.bundlingImage,
    command: [
      'bash', '-c',
      'GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /asset-output/bootstrap ./handler',
    ],
    user: 'root',
  };
}

function resolveStripeKey(scope: Construct): string {
  return secretsmanager.Secret.fromSecretNameV2(scope, 'StripeSecretKey', 'tour-commerce/stripe/secret-key')
    .secretValue.unsafeUnwrap();
}

function resolveWebhookSecret(scope: Construct): string {
  return secretsmanager.Secret.fromSecretNameV2(scope, 'StripeWebhookSecret', 'tour-commerce/stripe/webhook-secret')
    .secretValue.unsafeUnwrap();
}
