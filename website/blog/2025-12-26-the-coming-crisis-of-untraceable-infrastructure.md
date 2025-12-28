---
title: The Coming Crisis of Untraceable Infrastructure
authors: [rafasiqueira]
tags: [agentic-development, infrastructure-as-code, farseek, traceability]
description: Why agentic development needs a new approach to IaC.
hide_table_of_contents: false
---

Last week, I watched a demo of an AI agent provisioning AWS resources through a Model Context Protocol (MCP) server. In under a minute, it spun up an RDS instance, configured security groups, and deployed a Lambda function. The developer barely touched their keyboard.

It was impressive. It was also terrifying.

<!-- truncate -->

Because when that agent finished, there was no trace of what it built. No Terraform files. No CloudFormation templates. Nothing in Git. The infrastructure existed in AWS, but as far as the codebase was concerned, it didn't exist at all.

We're witnessing the explosive growth of agentic development. Cloud MCPs are making it trivially easy for AI agents to create infrastructure on the fly. And we're about to have a massive traceability problem.

This is why I built Farseek.

## The Ghost Infrastructure Problem

Here's the trajectory we're on:

AI coding assistants are getting remarkably good at understanding what infrastructure an application needs. "This app needs a PostgreSQL database, a Redis cache, and an S3 bucket for uploads"—an agent can figure this out from reading your code.

Cloud MCPs are giving these agents the ability to act on that understanding. Instead of writing Terraform and waiting for you to apply it, an agent can provision resources directly through AWS, GCP, or Azure APIs.

This is incredible for velocity. It's catastrophic for operations.

What happens when that developer leaves the company? When you need to audit your infrastructure for compliance? When you need to replicate the environment for a new team member?

And then there's the bill.

## The Cost Bleeding Problem

Here's a scenario that's about to become very common: an agent spins up a db.r5.2xlarge RDS instance to test a feature. The developer moves on to something else. The instance keeps running. For weeks. For months.

Multiply this across a team of developers, each working with AI agents that can provision infrastructure on demand, and you have a recipe for runaway cloud costs. Not from malice or negligence—just from the natural entropy of fast iteration without traceability.

The insidious part is that you won't notice immediately. Cloud bills are noisy. A few hundred dollars here, a few hundred there—it blends into the baseline. By the time someone asks "why did our AWS spend double this quarter?", you're spelunking through the console trying to match resources to projects, with no commit history to guide you.

Traditional cost management tools can tell you what is expensive. They can't tell you *why* it exists or whether it's still needed. That context lives in your codebase—or it should.

The speed that makes agentic infrastructure creation powerful is the same speed that makes it expensive. And telling agents to slow down and use traditional Terraform workflows defeats the entire purpose.

## Git as the Mandatory Checkpoint

Farseek takes a different approach: instead of choosing between speed and traceability, it makes traceability the fast path.

The core idea is simple. **Every infrastructure change—whether created by a human or an agent—must be declared in a .tf file and committed to Git.** But unlike traditional Terraform, applying those changes is nearly instantaneous because Farseek doesn't maintain a separate state file. It calculates what changed by comparing Git commits.

```bash
# Agent creates infrastructure by adding .tf files
# Then applies the change immediately
farseek apply --from HEAD~1 --to HEAD
```

The workflow becomes:

1.  Agent (or human) declares infrastructure in `.tf` files
2.  Changes are committed to Git
3.  Farseek applies the delta in seconds
4.  Git history becomes the complete audit trail

**No state file to corrupt. No remote backend to configure. No locking conflicts. Just Git commits and fast applies.**

## Why Not Just Use Terraform Faster?

I tried. The fundamental issue is that Terraform's architecture assumes the state file is the source of truth. Every operation requires reading the entire state, reconciling it with your declarations, and computing a plan.

For small projects, this takes seconds. For production infrastructure with hundreds of resources, it takes minutes. Ask an AI agent to wait 3 minutes between iterations, and you've killed the agentic development loop.

Users have been struggling with this for years. There's a feature request from 2015 asking for glob patterns in Terraform's `-target` flag—a workaround that would let users target multiple related resources without processing the entire state. Ten years and 650+ upvotes later, it's still open. The demand is clear: people need faster, more targeted operations. The architecture makes it hard to deliver.

Farseek inverts the model. **Your .tf files are the source of truth.** Git history tracks all changes. When you run `farseek plan`, it compares declarations between commits and queries cloud providers only for the specific resources that changed.

The result: plans that take seconds regardless of total infrastructure size. Fast enough that agents can iterate. Fast enough that the "just commit it" workflow doesn't feel like a tax.

## Traceability by Design

When infrastructure declarations live in Git alongside your application code, you get traceability for free:

*   **Every resource has a commit.** You can `git blame` any `.tf` file to see who (or what) created each resource and when.
*   **Every change has context.** The commit message, the PR discussion, the related code changes—it's all connected.
*   **Every environment is reproducible.** Check out any commit, run `farseek apply`, and you have that exact infrastructure state.
*   **Every agent action is auditable.** When an AI agent creates resources, it does so by committing `.tf` files. The audit trail is automatic.

Cost attribution becomes trivial. When your cloud bill spikes, you can trace every resource back to a commit. That mystery RDS instance? `git log` tells you it was created three months ago for a feature that never shipped. Delete the `.tf` file, run `farseek apply`, and it's gone.

This isn't just about compliance (though it helps). It's about maintainability and cost control. Six months from now, when you're trying to understand why a particular resource exists and whether you still need it, the answer is in your Git history.

## The Target User

I built Farseek for a specific context: startup teams where developers own their infrastructure.

At most startups, there's no dedicated platform team. The same engineers writing application code are also configuring databases, setting up networking, and debugging IAM policies. They don't have time to become Terraform experts. They definitely don't have time to wait for slow plan/apply cycles.

These teams are also the ones most likely to adopt agentic development tools. They're moving fast, experimenting constantly, and delegating more to AI assistants every month.

For these teams, Farseek offers:

1.  **One mental model.** Git is already your source of truth for code. Now it's your source of truth for infrastructure too. Your existing workflows—branches, PRs, code review—apply unchanged.
2.  **Speed that matches agentic iteration.** When an AI agent can create infrastructure as fast as it can write code, the feedback loop stays tight.
3.  **Traceability without process overhead.** You don't need to add extra steps or slow down. The audit trail is automatic.

## Building on Terraform, Not Against It

Farseek isn't a clean-room implementation. It's built on OpenTofu and deliberately reuses Terraform's parser, HCL syntax, and the entire provider ecosystem. If you've written Terraform before, you already know how to write Farseek configurations. Your existing `.tf` files work unchanged.

This is intentional. Terraform established HCL as the lingua franca of infrastructure-as-code. Thousands of providers exist. Millions of engineers know the syntax. We're not trying to fragment that ecosystem—we're offering a different runtime model for the same language.

Think of it like the relationship between Node.js and Deno, or between Docker and Podman. Same interfaces, different execution philosophy.

Today, cloud providers like GCP heavily recommend Terraform for infrastructure provisioning. My hope is that Farseek becomes a compelling alternative for providers to recommend—especially for use cases where speed and Git-native workflows matter more than traditional state management.

That said, Farseek isn't for everyone. If you have a dedicated platform team, complex state management requirements, and compliance frameworks built around Terraform's model, those investments still make sense.

Farseek is for teams where:

*   The overhead of state management exceeds its benefits
*   Speed of iteration matters more than enterprise features
*   Developers are adopting agentic tools and need infrastructure to keep pace
*   Terraform familiarity should transfer, not be discarded

## Getting Started

Farseek is available now as an alpha release, forked from OpenTofu and fully compatible with its provider ecosystem:

```bash
# Install Farseek
curl -sSL https://farseek.dev/install.sh | sh

# Initialize in an existing project
farseek init

# See what changed since your last deployment
farseek plan
```

The project is open source under MPL-2.0. I'd love contributions—bug reports, feature requests, documentation, or code.

## The Future Is Agentic and Auditable

We're at an inflection point. The tools that make agentic development powerful are arriving faster than the tools that make it safe and sustainable. Cloud MCPs will let AI agents provision infrastructure instantly. The question is whether that infrastructure will be traceable—or whether we'll spend the next decade cleaning up ghost resources and unexplained cloud bills.

Farseek is my bet on a specific future: one where agents and humans collaborate on infrastructure as naturally as they collaborate on code, where every resource is declared and versioned, and where Git history tells the complete story of how your systems evolved.

If that future resonates with you, I'd love to have you along for the ride.

Farseek is open source at [github.com/rafagsiqueira/farseek](https://github.com/rafagsiqueira/farseek). Star the repo, open an issue, or reach out—I'm building this in the open and want to hear what you think.
