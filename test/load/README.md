# Load and performance tests

Load tests aim to test the performance of the system under heavy load. For Agones, game server allocation is an example where heavy load and multiple parallel operations can be envisioned. Locust provides a good framework for testing a system under heavy load. It provides a light-weight mechanism to launch thousands of workers that run a given test.

The goal of performance tests is to provide metrics on various operations. For
Agones, fleet scaling is a good example where performance metrics are useful.
Similar to load tests, Locust can be used for performance tests with the main
difference being the number of workers that are launched.

