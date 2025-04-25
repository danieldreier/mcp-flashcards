# Testing

This document describes how to run and configure the different types of tests in this project.

## Property-Based Testing

Property-based tests use the `gopter` library to generate sequences of commands and verify system invariants. These tests are located in the `propertytest` directory.

### Running Property Tests

To run the property-based tests, use the standard Go test command, targeting the `propertytest` package:

```bash
go test ./propertytest/...
```

You can add the `-v` flag for more verbose output, which is helpful for seeing the progress and generated command sequences:

```bash
go test -v ./propertytest/...
```

### Configuring Property Tests

The property-based tests can be configured using environment variables:

1.  **Number of Test Sequences:**
    *   By default, the command sequence property test runs 100 sequences.
    *   To change the number of sequences, set the `MCP_PROPERTY_TEST_SEQUENCES` environment variable.
    *   Example: Run 500 test sequences:
        ```bash
        MCP_PROPERTY_TEST_SEQUENCES=500 go test -v ./propertytest/...
        ```

2.  **Deterministic Runs (Reproducing Failures):**
    *   When a property test fails, `gopter` will output the random seed used for that run.
    *   To re-run the exact same failing sequence, set the `MCP_PROPERTY_TEST_SEED` environment variable to the reported seed value.
    *   Example: Re-run with a specific seed (replace `1234567890` with the actual seed):
        ```bash
        MCP_PROPERTY_TEST_SEED=1234567890 go test -v ./propertytest/...
        ```
    *   This allows for deterministic reproduction and debugging of specific failures found during randomized testing. 