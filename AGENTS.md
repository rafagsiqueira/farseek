# Farseek AI Agents Information

Welcome to **Farseek**, a specialized fork of OpenTofu. This document provides essential context and instructions for AI agents working on this codebase.

## Project Mission

Farseek aims to simplify cloud infrastructure management by eliminating the need for a centralized state file. instead, it relies on Git history and selective cloud polling to calculate drift and forge execution plans.

## Core Architectural Decisions

### 1. State-less Planning
Farseek does **not** maintain a `.tfstate` file. The source of truth for the *intended* state is always the Git history.

### 2. Git-Driven Drift Calculation
- **Tracking Progress**: The last analyzed commit SHA is stored in a file named `.farseek_sha`.
- **Differential Analysis**: Instead of polling every resource in the cloud provider, Farseek analyzes the Git history from the commit in `.farseek_sha` to the current `HEAD`.
- **Selective Polling**: Only resources that have been modified (created, updated, or deleted) in the HCL files within that Git range are retrieved from the cloud provider for drift calculation.

### 3. Selective Forging
Execution plans are forged only for the subset of resources identified in the Git history analysis. This significantly reduces plan time and avoids unnecessary API calls to cloud providers.

## Development Workflow: Test-Driven Development (TDD)

We strictly follow a **Test-Driven Development (TDD)** approach for all new features and bug fixes. When working as an agent:

1.  **Write a Failing Test First**: Before implementing any logic, create a test case that captures the desired behavior and fails.
2.  **Implement Minimum Logic**: Write just enough code to make the test pass.
3.  **Refactor**: Clean up the code while ensuring tests remain green.
4.  **Verification**: Always run all tests before marking a task as complete.

### How to Run Tests

Farseek uses the standard Go testing toolchain, wrapped in a `Makefile` for convenience:

-   **Unit Tests**: `make test` runs `go test -v ./...`.
-   **Coverage**: `make test-with-coverage` generates a coverage report.
-   **Integration Tests**: See `Makefile` for specific targets like `test-s3`, `test-gcp`, etc.

## Agent Instructions

- **Analyze `.farseek_sha`**: When implementing plan/apply logic, always ensure you are reading and updating the `.farseek_sha` file correctly.
- **Git Integration**: Focus on using Git primitives to identify changed files and resource blocks.
- **Provider Efficiency**: Optimize provider calls by only requesting state for resources identified in the diff.
- **TDD Compliance**: All PRs and edits must include corresponding tests.
