import * as cdk from 'aws-cdk-lib';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as sns from 'aws-cdk-lib/aws-sns';
import * as subs from 'aws-cdk-lib/aws-sns-subscriptions';
import * as sqs from 'aws-cdk-lib/aws-sqs';
import * as iam from 'aws-cdk-lib/aws-iam';
import { Construct } from 'constructs';

export interface CommerceEventsStackProps extends cdk.StackProps {
  /** Lambda that handles incoming events (e.g. tour-agent booking confirmed). */
  ordersFunctionArn: string;
  /** Lambda that handles payment webhook events from SQS (DLQ retry). */
  paymentsFunctionArn: string;
}

export class CommerceEventsStack extends cdk.Stack {
  /** EventBridge bus for commerce domain events. */
  public readonly eventBus: events.EventBus;

  constructor(scope: Construct, id: string, props: CommerceEventsStackProps) {
    super(scope, id, props);

    // ── EventBridge bus ────────────────────────────────────────────────────────
    this.eventBus = new events.EventBus(this, 'CommerceBus', {
      eventBusName: 'tour-commerce',
    });

    const ordersFn = lambda.Function.fromFunctionArn(this, 'OrdersFn', props.ordersFunctionArn);
    const paymentsFn = lambda.Function.fromFunctionArn(this, 'PaymentsFn', props.paymentsFunctionArn);

    // ── Rule: tour-agent booking confirmed → update commerce order item ────────
    new events.Rule(this, 'BookingConfirmedRule', {
      eventBus: this.eventBus,
      ruleName: 'commerce-booking-confirmed',
      description: 'Fires when tour-agent confirms an excursion booking',
      eventPattern: {
        source: ['tour-agent.bookings'],
        detailType: ['BookingConfirmed'],
      },
      targets: [new targets.LambdaFunction(ordersFn, { retryAttempts: 2 })],
    });

    // ── Rule: payment succeeded → trigger post-purchase AI upsell ─────────────
    new events.Rule(this, 'PaymentSucceededRule', {
      eventBus: this.eventBus,
      ruleName: 'commerce-payment-succeeded',
      description: 'Fires when a Stripe payment succeeds so we can trigger AI upsell',
      eventPattern: {
        source: ['tour-commerce.payments'],
        detailType: ['PaymentSucceeded'],
      },
      targets: [new targets.LambdaFunction(paymentsFn, { retryAttempts: 2 })],
    });

    // ── Dead-letter queue for failed commerce events ───────────────────────────
    const dlq = new sqs.Queue(this, 'CommerceDLQ', {
      queueName: 'tour-commerce-dlq',
      retentionPeriod: cdk.Duration.days(14),
      encryption: sqs.QueueEncryption.SQS_MANAGED,
    });

    // ── SNS topic: order status notifications to subscribers ──────────────────
    const orderNotificationsTopic = new sns.Topic(this, 'OrderNotificationsTopic', {
      topicName: 'tour-commerce-order-notifications',
      displayName: 'Tour Commerce Order Notifications',
    });

    // Fan-out: future email/SMS subscriptions can be wired here.
    // e.g. orderNotificationsTopic.addSubscription(new subs.EmailSubscription('ops@example.com'));

    // Allow EventBridge to put messages on the DLQ.
    dlq.addToResourcePolicy(new iam.PolicyStatement({
      principals: [new iam.ServicePrincipal('events.amazonaws.com')],
      actions: ['sqs:SendMessage'],
      resources: [dlq.queueArn],
    }));

    // ── outputs ───────────────────────────────────────────────────────────────
    new cdk.CfnOutput(this, 'CommerceBusArn', {
      value:      this.eventBus.eventBusArn,
      exportName: 'TourCommerceEventBusArn',
    });
    new cdk.CfnOutput(this, 'CommerceDLQUrl', {
      value:      dlq.queueUrl,
      exportName: 'TourCommerceDLQUrl',
    });
    new cdk.CfnOutput(this, 'OrderNotificationsTopicArn', {
      value:      orderNotificationsTopic.topicArn,
      exportName: 'TourCommerceOrderNotificationsArn',
    });
  }
}
