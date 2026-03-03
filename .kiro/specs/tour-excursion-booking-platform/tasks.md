# Implementation Plan: Tour Excursion Booking Platform

## Overview

This implementation plan breaks down the AWS-based tour excursion booking platform into discrete, sequential coding tasks. The platform enables travelers to discover and book shore excursions with intelligent time-window filtering for cruise passengers and flight arrivals, real-time inventory management, dynamic pricing, and secure payment processing. The architecture leverages AWS serverless services including Lambda, API Gateway, Cognito, Aurora Serverless v2 PostgreSQL, ElastiCache Redis, and OpenSearch.

**Technology Stack:**
- Backend: Go (Golang) for all Lambda functions
- Frontend: React with TypeScript for UI components
- Infrastructure: AWS CDK with TypeScript
- Testing: Go testing framework with gopter or rapid for property-based tests

## Tasks

- [x] 1. Set up AWS infrastructure foundation and core services
  - [x] 1.1 Create core AWS infrastructure
    - Create AWS CDK project structure with TypeScript
    - Define VPC with public/private subnets, NAT Gateway, and security groups
    - Set up Aurora Serverless v2 PostgreSQL cluster with auto-scaling
    - Configure ElastiCache Redis cluster with encryption at rest
    - Configure S3 buckets for vouchers and static assets with encryption
    - Set up CloudFront CDN distribution
    - Set up Secrets Manager for API keys and database credentials
    - Configure KMS keys for encryption
    - Set up CloudWatch log groups and X-Ray tracing
    - _Requirements: NFR-1, NFR-2, NFR-3, NFR-4, 21.1, 21.2, 21.3_
  
  - [ ]* 1.2 Configure AWS WAF (optional - future backlog)
    - Configure AWS WAF with SQL injection and XSS protection rules
    - _Requirements: 22.4, 22.5_

- [x] 2. Create initial database schema
  - Create initial database schema SQL scripts
  - Create tables: users, destinations, excursions, excursion_schedules, suppliers, cruise_port_calls, flight_arrivals, bookings, payments, refunds, reviews, categories, inventory_records, pricing_configurations, sessions
  - Create sessions table with JSONB data column, expires_at timestamp, and indexes
  - Configure pg_cron extension for automated session cleanup
  - Create indexes on frequently queried fields (destination_id, date, cruise_line, excursion_id, session expires_at)
  - Set up foreign key constraints and cascading rules
  - Configure database connection pooling with RDS Proxy
  - _Requirements: 1.1, 1.6, 2.1, 2.2, 3.1, 4.1, 5.1, 7.1, 8.1, 14.1_


- [ ] 3. Implement Authentication Service with Cognito integration
  - [ ] 3.1 Configure Amazon Cognito User Pool with email and phone attributes
    - Set up user pool with password policy (8+ chars, complexity requirements)
    - Configure email verification workflow
    - Set up OAuth 2.0 app clients for web and mobile
    - Configure identity providers for Facebook, Google, Instagram, TikTok
    - Store OAuth client secrets in Secrets Manager
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 1.10_
  
  - [ ] 3.2 Implement Authentication Lambda functions in Go
    - Create signup Lambda with email/password registration (Go)
    - Create login Lambda with credential validation (Go)
    - Create social login Lambda with OAuth 2.0 flow (Go)
    - Create token refresh Lambda (Go)
    - Create password reset Lambda with email verification (Go)
    - Implement Cognito Lambda triggers (pre-signup, post-confirmation, pre-token generation) (Go)
    - Create user profile in RDS on successful authentication
    - _Requirements: 1.1, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9_
  
  - [ ]* 3.3 Write property test for authentication token generation using gopter or rapid
    - **Property 13: User Authentication - All authenticated API requests have valid JWT tokens**
    - **Validates: Requirements 1.4, 22.1, 22.2**
  
  - [ ]* 3.4 Write unit tests for authentication service using Go testing
    - Test email validation and uniqueness
    - Test password complexity requirements
    - Test social login token exchange
    - Test JWT token expiration
    - _Requirements: 1.1, 1.4, 1.9_

- [ ] 4. Implement API Gateway and authorization
  - Create REST API in API Gateway with CORS configuration
  - Configure Cognito authorizer for protected endpoints
  - Set up API Gateway request validation with JSON schemas
  - Configure throttling (10,000 requests/sec) and burst limits
  - Set up API Gateway caching with 5-minute TTL for search endpoints
  - Create API Gateway stages (dev, staging, production)
  - _Requirements: 22.1, 22.2, 22.3, 22.9, 26.9, NFR-1_


- [ ] 5. Implement Destination Service
  - [ ] 5.1 Create Destination Lambda functions in Go
    - Implement searchDestinations with OpenSearch full-text search (Go)
    - Implement getDestination with destination details (Go)
    - Implement getDestinationByPortCode for cruise integration (Go)
    - Implement getDestinationByAirportCode for flight integration (Go)
    - Implement listDestinations with pagination (Go)
    - _Requirements: 2.1, 2.2, 3.1, 4.4_
  
  - [ ] 5.2 Set up OpenSearch cluster and indexing
    - Create OpenSearch domain with encryption
    - Define destination index mapping with full-text fields
    - Implement indexing Lambda in Go to sync RDS data to OpenSearch
    - Configure autocomplete with edge n-grams
    - _Requirements: 2.1, 25.1, 25.8_
  
  - [ ]* 5.3 Write unit tests for destination service using Go testing
    - Test search with various filters
    - Test port code and airport code lookups
    - Test pagination
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [ ] 6. Implement Excursion Service with schedule management
  - [ ] 6.1 Create Excursion Lambda functions in Go
    - Implement searchExcursions with OpenSearch and filters (Go)
    - Implement getExcursion with full excursion details (Go)
    - Implement getExcursionSchedule for specific dates (Go)
    - Implement createExcursion and updateExcursion for admin (Go)
    - _Requirements: 2.3, 2.4, 2.5, 2.6, 2.7, 2.8_
  
  - [ ] 6.2 Implement excursion filtering logic
    - Create filterByCategory, filterByPrice, filterByDuration, filterByDifficulty functions
    - Implement multi-filter AND logic
    - Implement facet counting for filter options
    - _Requirements: 2.4, 25.2, 25.3, 25.4, 25.5, 25.6, 25.7_
  
  - [ ] 6.3 Index excursions in OpenSearch
    - Define excursion index mapping with nested schedules
    - Implement indexing Lambda in Go for excursion data
    - Include destination name, categories, supplier name in index
    - _Requirements: 2.3, 25.1, 25.10_
  
  - [ ]* 6.4 Write unit tests for excursion service using Go testing
    - Test search with multiple filters
    - Test schedule retrieval
    - Test blackout date handling
    - _Requirements: 2.3, 2.4, 2.5, 2.6_


- [ ] 7. Implement Cruise Schedule Service with time-window calculations
  - [ ] 7.1 Create Cruise Schedule Lambda functions in Go
    - Implement getPortCall to retrieve cruise port call details (Go)
    - Implement searchPortCalls with filters (cruise line, ship, destination, date range) (Go)
    - Implement calculateTimeWindow with arrival buffer (+1hr) and departure buffer (-1.5hr) (Go)
    - Implement importSchedules for admin to ingest cruise data (Go)
    - Handle timezone-aware date/time parsing using IANA timezones
    - _Requirements: 3.1, 3.2, 3.6, 3.8, 17.3, 17.7_
  
  - [ ] 7.2 Implement cruise excursion eligibility filtering
    - Create getExcursionsForCruise that filters by time window
    - Verify excursion start time >= earliest start (arrival + 1hr)
    - Verify excursion end time <= latest end (departure - 1.5hr)
    - Check excursion operates on the day of week
    - Exclude blackout dates
    - _Requirements: 3.3, 3.4, 3.5_
  
  - [ ]* 7.3 Write property test for time window calculations using gopter or rapid
    - **Property 3: Time Window Validity - Cruise time windows always have positive duration**
    - **Validates: Requirements 3.2, 17.3**
  
  - [ ]* 7.4 Write unit tests for cruise schedule service using Go testing
    - Test time window calculation with various arrival/departure times
    - Test timezone conversions
    - Test excursion filtering with time windows
    - Test day of week validation
    - _Requirements: 3.2, 3.3, 3.4, 3.5, 3.8_

- [ ] 8. Implement Flight Schedule Service with transfer buffer
  - [ ] 8.1 Create Flight Schedule Lambda functions in Go
    - Implement getFlightArrival to retrieve flight details (Go)
    - Implement searchFlightArrivals with filters (Go)
    - Implement calculateEarliestStart with 2-hour transfer buffer (Go)
    - Implement importFlightSchedules to ingest flight data from external APIs (Go)
    - Handle timezone-aware date/time parsing
    - _Requirements: 4.1, 4.2, 4.5, 4.6, 4.7, 17.4, 17.8_
  
  - [ ] 8.2 Implement flight excursion eligibility filtering
    - Create getExcursionsForFlight that filters by earliest start time
    - Verify excursion start time >= arrival time + 2 hours
    - Match excursions by destination airport code
    - _Requirements: 4.3, 4.4_
  
  - [ ]* 8.3 Write unit tests for flight schedule service using Go testing
    - Test earliest start calculation
    - Test excursion filtering by airport code
    - Test timezone handling
    - _Requirements: 4.1, 4.2, 4.3, 4.4_


- [ ] 9. Implement Inventory Service with Redis-based locking
  - [ ] 9.1 Create Inventory Lambda functions in Go
    - Implement checkAvailability to query Redis for real-time inventory (Go)
    - Implement lockInventory with Redis MULTI/EXEC atomic operations (Go)
    - Implement releaseLock to manually release inventory holds (Go)
    - Implement confirmBooking to move held spots to booked spots (Go)
    - Implement getInventoryStatus for admin dashboard (Go)
    - Implement updateCapacity for admin to adjust capacity (Go)
    - Implement setBlackoutDate to prevent bookings (Go)
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.10_
  
  - [ ] 9.2 Implement Redis inventory data structures
    - Create inventory:{excursion_id}:{date} hash with total_capacity, booked, held, available
    - Create hold:{booking_id} hash with TTL of 300 seconds
    - Implement automatic lock expiration with Redis TTL
    - Implement atomic decrement for available spots
    - _Requirements: 5.3, 5.4, 5.5, 5.8_
  
  - [ ] 9.3 Implement inventory consistency checks
    - Create Lambda in Go to sync Redis inventory with RDS persistent records
    - Implement CloudWatch alarm for inventory inconsistencies
    - Create reconciliation job to detect and fix discrepancies
    - _Requirements: 5.9, 19.9, 19.10_
  
  - [ ]* 9.4 Write property test for inventory consistency using gopter or rapid
    - **Property 1: Inventory Consistency - booked + held + available always equals total capacity**
    - **Validates: Requirements 5.2, 5.9, 19.1, 19.2, 19.9**
  
  - [ ]* 9.5 Write property test for no overbooking using gopter or rapid
    - **Property 2: No Overbooking - booked spots never exceed total capacity**
    - **Validates: Requirements 5.2, 19.2, 19.4, 19.5, 19.6**
  
  - [ ]* 9.6 Write unit tests for inventory service using Go testing
    - Test availability check with sufficient capacity
    - Test lock acquisition success and failure
    - Test concurrent lock attempts (race conditions)
    - Test lock expiration after TTL
    - Test inventory consistency maintenance
    - _Requirements: 5.1, 5.3, 5.4, 5.5, 5.8, 5.9_

- [ ] 10. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.


- [ ] 11. Implement Pricing Engine with dynamic pricing
  - [ ] 11.1 Create Pricing Lambda functions in Go
    - Implement calculatePrice with base, tiered, and dynamic pricing logic (Go)
    - Implement getPricingTiers to retrieve pricing configuration (Go)
    - Implement setPricingRule for admin to configure pricing (Go)
    - Implement setSeasonalPricing for admin to set seasonal multipliers (Go)
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8, 6.9, 6.10_
  
  - [ ] 11.2 Implement dynamic pricing factors
    - Implement demand-based multiplier using capacity utilization thresholds
    - Implement seasonal multiplier based on date ranges
    - Implement cruise line premium pricing multiplier
    - Implement advance booking discount based on days in advance
    - Implement last-minute discount based on hours until departure
    - Calculate final price multiplier by combining all factors
    - _Requirements: 6.3, 6.4, 6.5, 6.6, 6.7_
  
  - [ ] 11.3 Implement price breakdown calculation
    - Calculate adult, child, and infant prices separately
    - Apply passenger type multipliers for tiered pricing
    - Generate detailed price breakdown with description, quantity, unit price, total
    - Sum all passenger types to calculate total price
    - _Requirements: 6.2, 6.8, 6.9_
  
  - [ ]* 11.4 Write property test for price positivity using gopter or rapid
    - **Property 5: Price Positivity - Total price and multipliers are always positive**
    - **Validates: Requirements 6.8, 6.10**
  
  - [ ]* 11.5 Write unit tests for pricing engine using Go testing
    - Test fixed pricing calculation
    - Test tiered pricing with adult/child/infant
    - Test dynamic pricing with demand multiplier
    - Test seasonal pricing adjustments
    - Test cruise line premium pricing
    - Test advance booking and last-minute discounts
    - Test price breakdown summation
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8, 6.9_


- [ ] 12. Implement Payment Service with Stripe/Adyen integration
  - [ ] 12.1 Create Payment Lambda functions in Go
    - Implement createPaymentIntent with Stripe/Adyen SDK (Go)
    - Implement confirmPayment to verify payment status (Go)
    - Implement processRefund with refund amount and reason (Go)
    - Implement getPaymentStatus to query payment gateway (Go)
    - Store payment gateway credentials in Secrets Manager
    - _Requirements: 7.6, 7.7, 8.3, 8.4, 20.1, 20.2, 20.3, 20.7_
  
  - [ ] 12.2 Implement payment webhook handler
    - Create webhook Lambda in Go to handle payment gateway events
    - Verify webhook signature for security
    - Handle payment_intent.succeeded event
    - Handle payment_intent.failed event
    - Handle refund.completed event
    - Log all payment events for audit trail
    - _Requirements: 7.9, 7.10, 20.7, 30.3_
  
  - [ ] 12.3 Implement payment security measures
    - Use payment gateway client-side SDK for card tokenization
    - Never store credit card numbers in the system
    - Use TLS 1.2+ for all payment communications
    - Mask sensitive data in logs
    - Store payment intent IDs in PostgreSQL with encryption at rest
    - _Requirements: 20.1, 20.2, 20.3, 20.4, 20.5, 20.6, 20.8_
  
  - [ ]* 12.4 Write property test for payment idempotency using gopter or rapid
    - **Property 9: Payment Idempotency - Processing same payment intent multiple times results in single booking**
    - **Validates: Requirements 7.10, NFR-5**
  
  - [ ]* 12.5 Write unit tests for payment service using Go testing
    - Test payment intent creation
    - Test webhook signature verification
    - Test successful payment handling
    - Test failed payment handling
    - Test refund processing
    - _Requirements: 7.6, 7.7, 7.9, 7.10, 8.3, 8.4_


- [ ] 13. Implement Booking Service with complete lifecycle management
  - [ ] 13.1 Create Booking Lambda functions in Go
    - Implement createBooking with validation, inventory locking, and payment intent creation (Go)
    - Implement getBooking to retrieve booking details (Go)
    - Implement listUserBookings with filters and pagination (Go)
    - Implement cancelBooking with refund processing (Go)
    - Implement modifyBooking for date/passenger changes (Go)
    - Implement getVoucher to retrieve voucher PDF (Go)
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.8, 8.1, 8.2_
  
  - [ ] 13.2 Implement booking validation logic
    - Validate excursion is active and has available capacity
    - Validate cruise time window if cruisePortCallId provided
    - Validate flight earliest start if flightArrivalId provided
    - Validate booking cutoff (current time + cutoff hours <= departure time)
    - Validate passenger information completeness
    - Validate emergency contact phone number format (E.164)
    - _Requirements: 7.1, 7.2, 7.3, 15.1, 15.2, 18.2, 18.3, 18.4_
  
  - [ ] 13.3 Implement booking creation workflow
    - Check availability and validate eligibility
    - Lock inventory with 5-minute TTL
    - Calculate pricing with dynamic factors
    - Create payment intent
    - Create booking record with status PaymentHold
    - Return payment intent client secret to user
    - _Requirements: 7.4, 7.5, 7.6, 7.7, 7.8_
  
  - [ ] 13.4 Implement payment confirmation workflow
    - Handle payment webhook event
    - Update booking status to Confirmed
    - Update payment status to Paid
    - Move held inventory to booked inventory
    - Generate voucher PDF with QR code
    - Store voucher in S3 with signed URL
    - Publish BookingConfirmed event to EventBridge
    - Send booking confirmation to SQS queue
    - _Requirements: 7.10, 7.11, 7.12, 7.13, 7.14_
  
  - [ ]* 13.5 Write property test for booking atomicity using gopter or rapid
    - **Property 4: Booking Atomicity - Confirmed bookings have payment, inventory, and voucher**
    - **Validates: Requirements 7.10, 7.11, 7.12, 7.13**
  
  - [ ]* 13.6 Write property test for passenger count using gopter or rapid
    - **Property 6: Passenger Validation - Total passengers equals passenger array length**
    - **Validates: Requirements 7.1, 15.1**
  
  - [ ]* 13.7 Write unit tests for booking service using Go testing
    - Test booking creation with valid data
    - Test booking validation failures
    - Test time window validation
    - Test cutoff time enforcement
    - Test payment confirmation workflow
    - Test booking cancellation
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.10, 8.1, 18.3, 18.4_


- [ ] 14. Implement cancellation and refund processing
  - [ ] 14.1 Create cancellation Lambda functions in Go
    - Implement cancelBooking with cancellation policy calculation (Go)
    - Calculate cancellation fee based on excursion policy
    - Calculate refund amount (total price - cancellation fee)
    - Update booking status to Cancelled
    - Initiate refund with payment gateway
    - Release booked inventory back to available
    - Publish BookingCancelled event to EventBridge
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.6, 8.7, 8.8_
  
  - [ ] 14.2 Implement refund tracking
    - Create refund record with status Processing
    - Handle refund webhook events from payment gateway
    - Update refund status to Completed or Failed
    - Update booking payment status to Refunded or PartiallyRefunded
    - Send refund confirmation notification
    - _Requirements: 8.4, 8.5, 8.7, 8.9_
  
  - [ ]* 14.3 Write property test for refund constraints using gopter or rapid
    - **Property 12: Refund Constraint - Refund amount never exceeds original payment**
    - **Validates: Requirements 8.3, 8.4**
  
  - [ ]* 14.4 Write unit tests for cancellation service using Go testing
    - Test cancellation fee calculation
    - Test refund amount calculation
    - Test inventory release
    - Test refund processing
    - _Requirements: 8.1, 8.2, 8.3, 8.6_

- [ ] 15. Implement voucher generation with QR codes
  - [ ] 15.1 Create voucher generation Lambda in Go
    - Generate PDF voucher with booking details using Go PDF library (e.g., gofpdf)
    - Include excursion title, date, start time, meeting point, passenger names, booking ID
    - Generate QR code containing booking ID using Go QR library (e.g., go-qrcode)
    - Embed QR code image in PDF
    - Upload PDF to S3 with encryption
    - Generate signed URL with 1-year expiration
    - Update booking record with voucher URL
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5, 16.6_
  
  - [ ] 15.2 Implement QR code scanning for check-in
    - Create Lambda in Go to retrieve booking details from QR code
    - Update booking check-in status
    - Return passenger information for supplier verification
    - _Requirements: 16.8_
  
  - [ ]* 15.3 Write property test for voucher accessibility using gopter or rapid
    - **Property 15: Voucher Accessibility - Confirmed bookings have accessible vouchers with valid QR codes**
    - **Validates: Requirements 16.5, 16.6, 16.8**
  
  - [ ]* 15.4 Write unit tests for voucher generation using Go testing
    - Test PDF generation with booking data
    - Test QR code generation
    - Test S3 upload and signed URL generation
    - Test voucher retry on failure
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5, 16.9_

- [ ] 16. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.


- [ ] 17. Implement event-driven notification system
  - [ ] 17.1 Set up EventBridge, SQS, and SNS infrastructure
    - Create EventBridge event bus for booking events
    - Create SQS queues: booking-confirmations, supplier-notifications, booking-reminders
    - Create SNS topics for email and SMS notifications
    - Configure dead-letter queues for failed messages
    - Set up SQS Lambda triggers with batch processing
    - _Requirements: 23.1, 23.2, 23.3, 23.4, 23.5, 23.6, 23.7, 23.10_
  
  - [ ] 17.2 Implement notification Lambda functions in Go
    - Create email notification Lambda using Amazon SES (Go)
    - Create SMS notification Lambda using Amazon Pinpoint (Go)
    - Implement booking confirmation email with voucher attachment
    - Implement supplier notification with passenger manifest
    - Implement booking reminder email (24 hours before departure)
    - Implement cancellation confirmation email
    - Implement refund confirmation email
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.9, 9.10_
  
  - [ ] 17.3 Implement event publishing
    - Publish BookingConfirmed event on successful payment
    - Publish BookingCancelled event on cancellation
    - Publish InventoryUpdated event on capacity changes
    - Publish PaymentCompleted event on payment success
    - Include all relevant context in events (booking ID, user ID, excursion ID, timestamp)
    - _Requirements: 23.1, 23.2, 23.3, 23.8, 23.9, 23.10_
  
  - [ ] 17.4 Implement retry logic with exponential backoff
    - Configure SQS message retry with exponential backoff
    - Set maximum retry attempts (3-5 retries)
    - Move failed messages to dead-letter queue
    - Create CloudWatch alarm for dead-letter queue depth
    - _Requirements: 23.6, 23.7_
  
  - [ ]* 17.5 Write unit tests for notification system using Go testing
    - Test email sending with SES
    - Test SMS sending with Pinpoint
    - Test event publishing to EventBridge
    - Test SQS message processing
    - Test retry logic and dead-letter queue
    - _Requirements: 9.1, 9.2, 9.8, 23.4, 23.5, 23.6_


- [ ] 18. Implement timezone handling across all services
  - [ ] 18.1 Create timezone utility functions in Go
    - Implement parseDateTime with IANA timezone support using Go time package
    - Implement convertToUTC for database storage
    - Implement convertToDestinationTimezone for display
    - Implement calculateHoursDifference in destination timezone
    - Implement validateTimezone to check IANA timezone validity
    - _Requirements: 17.1, 17.2, 17.3, 17.4, 17.5, 17.6, 17.7, 17.8_
  
  - [ ] 18.2 Apply timezone handling to all date/time operations
    - Store all timestamps in UTC in database
    - Convert to destination timezone for time window calculations
    - Convert to destination timezone for cutoff validation
    - Parse cruise arrival/departure times with port timezone
    - Parse flight arrival times with airport timezone
    - _Requirements: 17.1, 17.2, 17.3, 17.4, 17.5, 17.7, 17.8_
  
  - [ ]* 18.3 Write property test for timezone consistency using gopter or rapid
    - **Property 10: Timezone Consistency - All timestamps stored in UTC and displayed in destination timezone**
    - **Validates: Requirements 17.1, 17.2**
  
  - [ ]* 18.4 Write unit tests for timezone handling using Go testing
    - Test UTC storage and retrieval
    - Test timezone conversion
    - Test time window calculations across timezones
    - Test cutoff validation with different timezones
    - _Requirements: 17.1, 17.2, 17.3, 17.4, 17.5_

- [ ] 19. Implement booking cutoff enforcement
  - [ ] 19.1 Create cutoff validation functions
    - Implement validateBookingCutoff to check hours until departure
    - Calculate hours difference in destination timezone
    - Compare with excursion schedule cutoff hours
    - Return validation error if within cutoff window
    - _Requirements: 18.1, 18.2, 18.3, 18.4_
  
  - [ ] 19.2 Apply cutoff filtering to search results
    - Filter out excursions where current time is within cutoff window
    - Display "Booking closed" message for excursions past cutoff
    - _Requirements: 18.5_
  
  - [ ]* 19.3 Write property test for booking cutoff using gopter or rapid
    - **Property 11: Booking Cutoff - All bookings respect minimum cutoff hours**
    - **Validates: Requirements 18.2, 18.3, 18.4**
  
  - [ ]* 19.4 Write unit tests for cutoff enforcement using Go testing
    - Test cutoff validation with various time differences
    - Test cutoff filtering in search results
    - Test cutoff updates by supplier
    - _Requirements: 18.2, 18.3, 18.4, 18.5, 18.6_


- [ ] 20. Implement Admin Service for suppliers and operations
  - [ ] 20.1 Create Supplier Dashboard Lambda functions in Go
    - Implement getSupplierDashboard with metrics (active excursions, bookings, revenue) (Go)
    - Implement manageExcursion for CRUD operations (Go)
    - Implement setBlackoutDates to prevent bookings (Go)
    - Implement getBookingManifest with passenger list and emergency contacts (Go)
    - Implement exportManifest to generate PDF manifest (Go)
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 10.7, 10.9_
  
  - [ ] 20.2 Create Operations Dashboard Lambda functions in Go
    - Implement getOperationsDashboard with daily metrics (Go)
    - Implement getPassengerList for specific excursion and date (Go)
    - Implement trackCapacity with real-time utilization (Go)
    - Display upcoming departures within 24 hours
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.10_
  
  - [ ] 20.3 Create Reporting Lambda functions in Go
    - Implement getRevenueReport aggregated by destination, cruise line, category (Go)
    - Implement getCommissionReport for supplier payments (Go)
    - Implement getRefundReport with amounts and statuses (Go)
    - Implement exportReport to generate CSV/PDF files (Go)
    - Store reports in S3 with signed URLs
    - _Requirements: 11.5, 11.6, 11.7, 11.8_
  
  - [ ]* 20.4 Write unit tests for admin service using Go testing
    - Test supplier dashboard metrics calculation
    - Test manifest generation with passenger data
    - Test revenue report aggregation
    - Test commission calculation
    - _Requirements: 10.1, 10.5, 11.1, 11.5, 11.6_

- [ ] 21. Implement User Profile Management
  - [ ] 21.1 Create User Profile Lambda functions in Go
    - Implement getUserProfile to retrieve user data (Go)
    - Implement updateUserProfile with validation (Go)
    - Implement updateEmail with verification workflow (Go)
    - Implement getBookingHistory with filters (Go)
    - Implement updateNotificationPreferences (Go)
    - Implement disconnectSocialAccount (Go)
    - Implement deleteAccount with GDPR compliance (Go)
    - Implement exportUserData for GDPR data portability (Go)
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5, 14.6, 14.7, 14.8, 14.9, 14.10_
  
  - [ ]* 21.2 Write unit tests for user profile service using Go testing
    - Test profile updates
    - Test email verification
    - Test booking history retrieval
    - Test account deletion
    - Test data export
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.9, 14.10_


- [ ] 22. Implement Review and Rating System
  - [ ] 22.1 Create Review Lambda functions in Go
    - Implement submitReview with validation (completed booking required) (Go)
    - Implement getReviews for excursion with pagination (Go)
    - Implement voteHelpful to increment helpful count (Go)
    - Implement addSupplierResponse for supplier replies (Go)
    - Validate user can only review once per booking
    - Mark reviews as verified purchase
    - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7_
  
  - [ ] 22.2 Implement rating calculation
    - Recalculate excursion average rating on new review
    - Update review count
    - Store ratings in excursion record for fast retrieval
    - _Requirements: 13.1, 13.9_
  
  - [ ]* 22.3 Write unit tests for review system using Go testing
    - Test review submission validation
    - Test one review per booking constraint
    - Test rating calculation
    - Test helpful votes
    - _Requirements: 13.3, 13.4, 13.5, 13.8, 13.9_

- [ ] 23. Implement AI-powered Recommendation Engine
  - [ ] 23.1 Create Recommendation Lambda functions in Go
    - Implement getRecommendations with user booking history (Go)
    - Implement getCrossSellRecommendations for complementary excursions (Go)
    - Implement getUpsellRecommendations for premium alternatives (Go)
    - Implement getTrendingExcursions by destination (Go)
    - Filter recommendations by time window if cruise/flight context exists
    - Filter recommendations by availability
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.7, 12.8_
  
  - [ ] 23.2 Implement collaborative filtering
    - Query user booking history from RDS
    - Find similar users based on booking patterns
    - Recommend excursions booked by similar users
    - _Requirements: 12.1, 12.5_
  
  - [ ] 23.3 Implement content-based filtering
    - Analyze excursion categories, difficulty, price range
    - Find similar excursions based on attributes
    - Recommend based on user preferences
    - _Requirements: 12.6_
  
  - [ ]* 23.4 Write unit tests for recommendation engine using Go testing
    - Test personalized recommendations
    - Test cross-sell logic
    - Test upsell logic
    - Test time window filtering
    - _Requirements: 12.1, 12.2, 12.3, 12.7_


- [ ] 24. Implement weather cancellation and force majeure handling
  - [ ] 24.1 Create supplier cancellation Lambda functions in Go
    - Implement cancelExcursionBySupplier with reason (weather, force majeure, operational) (Go)
    - Update all affected bookings to Cancelled status
    - Initiate full refunds for all affected bookings (waive cancellation fees)
    - Release booked inventory back to available
    - Send notification emails to all affected users
    - Prioritize notifications for cancellations within 24 hours
    - _Requirements: 29.1, 29.2, 29.3, 29.4, 29.5, 29.6, 29.7_
  
  - [ ]* 24.2 Write unit tests for supplier cancellation using Go testing
    - Test bulk booking cancellation
    - Test full refund processing
    - Test inventory release
    - Test notification sending
    - _Requirements: 29.1, 29.2, 29.3, 29.4_

- [ ] 25. Implement audit trail and compliance logging
  - [ ] 25.1 Create audit logging functions
    - Log booking creation with user ID, timestamp, excursion ID, details
    - Log booking status changes with old/new status, timestamp, actor
    - Log payment processing with payment intent ID, amount, currency, status
    - Log refund processing with refund ID, amount, reason
    - Log inventory updates with old/new capacity, timestamp, actor
    - Log personal data access with accessor, timestamp, data accessed
    - Log personal data deletion with timestamp (retain audit record)
    - _Requirements: 30.1, 30.2, 30.3, 30.4, 30.5, 30.6, 30.7_
  
  - [ ] 25.2 Set up CloudTrail for AWS API logging
    - Enable CloudTrail for all AWS API calls
    - Configure log retention for 7 years
    - Set up S3 bucket for long-term audit log storage
    - _Requirements: 30.8, 30.9_
  
  - [ ] 25.3 Implement audit log query interface
    - Create Lambda in Go to search audit logs by user, action, timestamp, resource
    - Implement filtering and pagination
    - _Requirements: 30.10_
  
  - [ ]* 25.4 Write unit tests for audit logging using Go testing
    - Test booking audit trail
    - Test payment audit trail
    - Test data access logging
    - _Requirements: 30.1, 30.2, 30.3, 30.6_

- [ ] 26. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.


- [ ] 27. Implement monitoring, observability, and alerting
  - [ ] 27.1 Set up CloudWatch metrics and dashboards
    - Create custom metrics for booking conversion rate, inventory utilization, payment success rate
    - Create CloudWatch dashboard with key metrics (Lambda duration, API latency, RDS connections, Redis hit rate)
    - Configure metric filters for error patterns
    - _Requirements: 24.1, 24.2, 24.3, 24.4, 24.10_
  
  - [ ] 27.2 Configure CloudWatch alarms
    - Create alarm for error rate > 1%
    - Create alarm for API latency exceeding SLA thresholds
    - Create alarm for inventory inconsistencies
    - Create alarm for payment success rate < 95%
    - Create alarm for dead-letter queue depth
    - Configure SNS topic for alarm notifications
    - Integrate with PagerDuty for on-call alerting
    - _Requirements: 24.5, 24.6, 24.7, 24.8, 27.10_
  
  - [ ] 27.3 Set up X-Ray distributed tracing
    - Enable X-Ray tracing for API Gateway
    - Enable X-Ray tracing for Go Lambda functions
    - Instrument critical code paths with X-Ray segments in Go
    - Create X-Ray service map for end-to-end request flow
    - _Requirements: 24.9_
  
  - [ ] 27.4 Configure structured logging
    - Implement structured JSON logging for all Go Lambda functions
    - Include correlation IDs for request tracing
    - Log errors with stack traces
    - Configure log retention (30 days for application logs)
    - _Requirements: 24.1, NFR-7_
  
  - [ ]* 27.5 Write unit tests for monitoring using Go testing
    - Test custom metric publishing
    - Test alarm triggering conditions
    - Test structured log formatting
    - _Requirements: 24.1, 24.10_


- [ ] 28. Implement API security and rate limiting
  - [ ] 28.1 Configure API Gateway security
    - Set up Cognito authorizer for protected endpoints
    - Implement JWT token validation on every request
    - Configure CORS with appropriate origins
    - Set up request throttling (10,000 requests/sec per account)
    - Configure burst limits (5,000 requests)
    - _Requirements: 22.1, 22.2, 22.9_
  
  - [ ]* 28.2 Configure AWS WAF rules (optional - future backlog)
    - Create WAF web ACL for API Gateway
    - Add SQL injection protection rule
    - Add XSS protection rule
    - Add rate limiting rule (1,000 requests per 5 minutes per IP)
    - Add geo-blocking rules if needed
    - _Requirements: 22.4, 22.5, 22.6_
  
  - [ ] 28.3 Implement input validation and sanitization
    - Create JSON schema validators for all API endpoints
    - Implement input sanitization to prevent XSS
    - Use parameterized queries for all database operations
    - Validate all user inputs before processing
    - _Requirements: 22.7, 22.8_
  
  - [ ] 28.4 Implement role-based access control
    - Define IAM roles for User, Supplier, Admin
    - Implement resource-level permissions (users access only their bookings)
    - Return HTTP 403 Forbidden for unauthorized access
    - _Requirements: 22.3, NFR-4_
  
  - [ ]* 28.5 Write unit tests for API security using Go testing
    - Test JWT token validation
    - Test unauthorized access rejection
    - Test input validation
    - Test SQL injection prevention
    - _Requirements: 22.1, 22.2, 22.3, 22.7, 22.8_

- [ ] 29. Implement data encryption and privacy controls
  - [ ] 29.1 Configure encryption at rest
    - Enable AES-256 encryption for Aurora Serverless v2 with KMS
    - Enable encryption for ElastiCache Redis
    - Enable server-side encryption for S3 buckets (SSE-KMS)
    - _Requirements: 21.1, 21.2, 21.3_
  
  - [ ] 29.2 Configure encryption in transit
    - Enforce TLS 1.2+ for all API communications
    - Configure HTTPS-only for CloudFront (redirect HTTP to HTTPS)
    - Use VPC endpoints for AWS service communication
    - _Requirements: 21.5, NFR-4_
  
  - [ ] 29.3 Implement application-level encryption for sensitive data
    - Encrypt passport numbers before storing in database
    - Encrypt bank account details for suppliers
    - Use AWS KMS for encryption key management
    - _Requirements: 21.6, 21.7_
  
  - [ ] 29.4 Implement data masking in logs
    - Mask email addresses in application logs
    - Mask phone numbers in application logs
    - Never log credit card data
    - _Requirements: 21.8, 20.6_
  
  - [ ] 29.5 Implement GDPR compliance features
    - Implement account deletion with permanent data erasure
    - Implement data export in JSON format
    - Ensure encrypted fields are deleted on account deletion
    - _Requirements: 21.9, 21.10, 14.9, 14.10_
  
  - [ ]* 29.6 Write unit tests for encryption and privacy using Go testing
    - Test data encryption/decryption
    - Test log masking
    - Test account deletion completeness
    - _Requirements: 21.6, 21.7, 21.8, 21.9_


- [ ] 30. Implement performance optimizations
  - [ ] 30.1 Configure caching layers
    - Set up ElastiCache Redis for inventory (sub-millisecond reads)
    - Configure CloudFront CDN for static assets with cache headers
    - Set up API Gateway caching for destination/excursion lists (TTL: 5 minutes)
    - Use PostgreSQL with JSONB for session data with indexed queries
    - Configure pg_cron for automated session cleanup based on expires_at
    - _Requirements: 26.1, 26.2, 26.8, NFR-1_
  
  - [ ] 30.2 Optimize database queries
    - Create indexes on destination_id, date, cruise_line, excursion_id
    - Configure Aurora Serverless v2 read replicas for search queries (if needed)
    - Implement connection pooling with RDS Proxy
    - Partition large tables (bookings by date)
    - _Requirements: 26.7, NFR-1_
  
  - [ ] 30.3 Configure Lambda optimizations
    - Set up provisioned concurrency for critical Go Lambda functions (booking, payment)
    - Tune Lambda memory allocation (1024MB-3008MB based on profiling)
    - Implement code bundling for Go binaries
    - Use Lambda layers for shared Go dependencies
    - _Requirements: 26.6, NFR-2_
  
  - [ ] 30.4 Implement asynchronous processing
    - Use SQS for non-critical operations (email, reporting) with Go Lambda consumers
    - Use EventBridge for event-driven workflows
    - Use Lambda async invocation for voucher generation
    - Implement batch processing for supplier manifests in Go
    - _Requirements: 26.10, NFR-2_
  
  - [ ]* 30.5 Write performance tests using Go testing
    - Test search latency under load
    - Test availability check latency
    - Test booking creation latency
    - Test concurrent booking scenarios
    - _Requirements: 26.1, 26.2, 26.3, 26.5_

- [ ] 31. Implement disaster recovery and high availability
  - [ ] 31.1 Configure backup and recovery
    - Enable Aurora Serverless v2 automated daily backups with 7-day retention
    - Enable Aurora Serverless v2 point-in-time recovery
    - Enable S3 versioning for vouchers
    - Configure cross-region replication for S3
    - _Requirements: 27.3, 27.4, 27.5_
  
  - [ ] 31.2 Configure high availability
    - Deploy Lambda functions across multiple availability zones
    - Configure Aurora Serverless v2 with automatic failover within region
    - Configure ElastiCache Redis cluster mode with automatic failover
    - Set up multi-region active-passive deployment
    - _Requirements: 27.1, 27.6, 27.7, 27.8, 27.9, NFR-3_
  
  - [ ] 31.3 Set up disaster recovery procedures
    - Document RTO (1 hour) and RPO (5 minutes) targets
    - Create runbooks for failover procedures
    - Configure SNS notifications for critical alarms
    - Integrate with PagerDuty for on-call team
    - _Requirements: 27.1, 27.2, 27.10_
  
  - [ ]* 31.4 Write disaster recovery tests using Go testing
    - Test database backup and restore
    - Test failover procedures
    - Test cross-region replication
    - _Requirements: 27.1, 27.2, 27.3, 27.4_


- [ ] 32. Implement external API integrations
  - [ ] 32.1 Integrate cruise schedule APIs
    - Implement Lambda in Go to fetch cruise schedules from external providers
    - Parse cruise schedule data and store in RDS
    - Implement caching strategy (24 hours)
    - Handle API failures with cached data fallback
    - Implement retry logic with exponential backoff
    - Set API timeout to 10 seconds
    - _Requirements: 3.7, 7.1_
  
  - [ ] 32.2 Integrate flight schedule APIs
    - Implement Lambda in Go to fetch flight data from FlightAware/Amadeus
    - Parse flight arrival data and store in RDS
    - Implement caching strategy (1 hour)
    - Handle API failures gracefully
    - Update flight status on delays/cancellations
    - _Requirements: 4.5, 4.6_
  
  - [ ] 32.3 Implement circuit breaker pattern
    - Create circuit breaker for external API calls
    - Open circuit after consecutive failures
    - Implement half-open state for recovery testing
    - Gracefully degrade when APIs unavailable
    - _Requirements: NFR-5_
  
  - [ ]* 32.4 Write integration tests for external APIs using Go testing
    - Test cruise schedule ingestion
    - Test flight schedule ingestion
    - Test caching behavior
    - Test circuit breaker functionality
    - _Requirements: 3.7, 4.5_

- [ ] 33. Implement accessibility and internationalization
  - [ ] 33.1 Implement multi-language support
    - Create language resource files (English, Spanish, French, German)
    - Implement language selection in user preferences (backend Go)
    - Display all UI text in selected language (frontend React/TypeScript)
    - _Requirements: 28.1_
  
  - [ ] 33.2 Implement locale-aware formatting
    - Format currency according to user locale (backend Go)
    - Format dates according to user locale (backend Go)
    - _Requirements: 28.2, 28.3_
  
  - [ ] 33.3 Implement accessibility features
    - Display wheelchair accessibility prominently (frontend React/TypeScript)
    - Allow filtering by accessibility features (backend Go + frontend React/TypeScript)
    - Ensure proper ARIA labels for screen readers (frontend React/TypeScript)
    - Maintain WCAG 2.1 AA color contrast ratios (frontend React/TypeScript)
    - Support keyboard navigation (frontend React/TypeScript)
    - _Requirements: 28.4, 28.5, 28.6, 28.7, 28.8_
  
  - [ ]* 33.4 Write unit tests for internationalization using Go testing
    - Test language selection
    - Test currency formatting
    - Test date formatting
    - _Requirements: 28.1, 28.2, 28.3_


- [ ] 34. Implement emergency contact and special requests handling
  - [ ] 34.1 Create emergency contact validation
    - Validate emergency contact name, phone, relationship are required
    - Validate phone number is in E.164 format
    - Encrypt emergency contact phone numbers at rest
    - _Requirements: 15.1, 15.2, 15.6_
  
  - [ ] 34.2 Implement special requests handling
    - Allow text field for special requests during booking
    - Display special requests in supplier manifest
    - Prominently display dietary restrictions and medical conditions
    - _Requirements: 15.3, 15.4, 15.5, 15.7_
  
  - [ ]* 34.3 Write unit tests for emergency contact handling using Go testing
    - Test phone number validation
    - Test E.164 format enforcement
    - Test encryption of phone numbers
    - _Requirements: 15.1, 15.2, 15.6_

- [ ] 35. Set up CI/CD pipeline and deployment
  - [ ] 35.1 Configure CI/CD pipeline
    - Set up GitHub/GitLab repository
    - Configure AWS CodeBuild for build stage
    - Configure automated test execution (unit, integration, property-based)
    - Set up AWS CodePipeline for deployment
    - Configure AWS CodeDeploy for Lambda deployments
    - _Requirements: NFR-6_
  
  - [ ] 35.2 Implement blue-green deployment
    - Configure Lambda aliases for traffic shifting
    - Set up API Gateway stages (blue/green)
    - Implement gradual rollout (10% → 50% → 100%)
    - Configure automatic rollback on CloudWatch alarms
    - _Requirements: NFR-6_
  
  - [ ] 35.3 Create deployment environments
    - Set up development environment with LocalStack
    - Set up staging environment in AWS
    - Set up production environment with multi-region support
    - Configure environment-specific variables
    - _Requirements: NFR-6_
  
  - [ ]* 35.4 Write deployment tests using Go testing
    - Test infrastructure provisioning
    - Test Lambda deployment
    - Test API Gateway configuration
    - Test rollback procedures
    - _Requirements: NFR-6_

- [ ] 36. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.


- [ ] 37. Integration testing and end-to-end validation
  - [ ]* 37.1 Write integration test for complete booking flow
    - Test user authentication → excursion search → booking creation → payment → confirmation
    - Verify inventory locked → payment processed → inventory updated → voucher generated → emails sent
    - _Requirements: 1.4, 2.3, 7.1, 7.10, 7.11, 7.12, 7.13, 7.14_
  
  - [ ]* 37.2 Write integration test for cruise schedule integration
    - Test cruise schedule ingestion → time window calculation → excursion filtering
    - Verify timezone handling and eligibility filtering
    - _Requirements: 3.1, 3.2, 3.3, 3.8_
  
  - [ ]* 37.3 Write integration test for payment gateway
    - Test payment intent creation → successful payment → webhook handling → booking confirmation
    - Test payment failure scenarios
    - _Requirements: 7.6, 7.7, 7.9, 7.10_
  
  - [ ]* 37.4 Write integration test for event-driven workflows
    - Test booking confirmed → EventBridge → SNS → SES (email sent)
    - Test booking confirmed → SQS → Lambda → supplier notification
    - _Requirements: 9.1, 9.2, 23.1, 23.2_
  
  - [ ]* 37.5 Write integration test for Redis inventory locking
    - Test concurrent booking attempts
    - Test lock expiration
    - Test inventory consistency under load
    - _Requirements: 5.3, 5.4, 5.5, 5.8, 19.6_

- [ ] 38. Documentation and API specification
  - Create OpenAPI specification for all API endpoints
  - Document authentication flows
  - Document error codes and responses
  - Create developer guide for API integration
  - Document deployment procedures
  - Create operations runbook
  - _Requirements: NFR-6_

- [ ] 39. Final system validation and deployment
  - Run full test suite (unit, integration, property-based)
  - Perform load testing to validate performance targets
  - Validate security controls (WAF, encryption, authentication)
  - Deploy to staging environment
  - Perform user acceptance testing
  - Deploy to production with gradual rollout
  - Monitor CloudWatch metrics and alarms
  - _Requirements: NFR-1, NFR-2, NFR-3, NFR-4, NFR-5_

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties using gopter or rapid
- Unit tests validate specific examples and edge cases using Go testing
- Integration tests validate end-to-end workflows
- Backend Lambda functions are written in Go (Golang)
- Frontend UI components are written in React with TypeScript
- Infrastructure code uses AWS CDK with TypeScript
- Follow AWS best practices for serverless architecture
- Ensure all sensitive data is encrypted at rest and in transit
- Implement comprehensive error handling and logging
- Use infrastructure as code (AWS CDK) for all AWS resources

