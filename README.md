# Farseek

- [HomePage](https://farseek.org/)
- [How to install](https://farseek.org/docs/intro/install)

Farseek is a specialized fork of OpenTofu that uses Git history to calculate cloud drift and forge plans without a state file.

The key features of Farseek are:

- **Infrastructure as Code**: Infrastructure is described using a high-level configuration syntax. This allows a blueprint of your datacenter to be versioned and treated as you would any other code. Additionally, infrastructure can be shared and re-used.

- **Execution Plans**: OpenTofu has a "planning" step where it generates an execution plan. The execution plan shows what OpenTofu will do when you call apply. This lets you avoid any surprises when OpenTofu manipulates infrastructure.

- **Resource Graph**: OpenTofu builds a graph of all your resources, and parallelizes the creation and modification of any non-dependent resources. Because of this, OpenTofu builds infrastructure as efficiently as possible, and operators get insight into dependencies in their infrastructure.

- **Change Automation**: Complex changesets can be applied to your infrastructure with minimal human interaction. With the previously mentioned execution plan and resource graph, you know exactly what OpenTofu will change and in what order, avoiding many possible human errors.

## Getting help and contributing

- Have a question?
  - Open a [GitHub issue](https://github.com/rafagsiqueira/farseek/issues/new/choose)
- Want to contribute?
  - Please read the [Contribution Guide](CONTRIBUTING.md).


## Reporting security vulnerabilities
If you've found a vulnerability or a potential vulnerability in Farseek please follow [Security Policy](https://github.com/rafagsiqueira/farseek/security/policy). We'll send a confirmation email to acknowledge your report, and we'll send an additional email when we've identified the issue positively or negatively.

## Reporting possible copyright issues

If you believe you have found any possible copyright or intellectual property issues, please contact liaison@farseek.org. We'll send a confirmation email to acknowledge your report.

## License

[Mozilla Public License v2.0](https://github.com/rafagsiqueira/farseek/blob/main/LICENSE)
