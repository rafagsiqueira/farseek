import React from "react";
import { IDE } from "../IDE";


type FeatureShowcaseProps = {
  title: string;
  description: string;
  version: string;
  codeExample: string;
  align: "left" | "right";
  color: string;
  filename?: string;
  docsUrl?: string;
};

function FeatureShowcase({
  title,
  description,
  version,
  codeExample,
  align,
  color,
  filename = "main.tf",
  docsUrl,
}: FeatureShowcaseProps) {
  const isRight = align === "right";

  // Hardcoded for Farseek as we only have the current docs for now
  const latestVersion = "1.0";
  const versionHref = "/docs/intro/whats-new/";

  return (
    <div className="flex flex-col lg:flex-row gap-8 mb-20 items-center">
      {/* Code example side */}
      <div
        className={`lg:w-1/2 w-full order-2 ${
          isRight ? "lg:order-2" : "lg:order-1"
        }`}
      >
        <IDE code={codeExample} language="hcl" filename={filename} />
      </div>

      {/* Content side */}
      <div
        className={`lg:w-1/2 w-full order-1 ${
          isRight ? "lg:order-1" : "lg:order-2"
        }`}
      >
        <div className="flex flex-wrap items-center gap-3 mb-3">
          <h3 className="text-2xl font-bold">{title}</h3>
          <a
            href={versionHref}
            className={`inline-flex px-3 py-1 text-sm font-medium rounded-full hover:opacity-90 transition-opacity ${color} hover:text-white`}
          >
            {`v${version}`}
          </a>
        </div>
        <p className="text-gray-600 dark:text-gray-400 text-lg mb-4">
          {description}
        </p>
        {docsUrl && (
          <a
            href={docsUrl}
            className="inline-flex items-center text-blue-600 dark:text-blue-400 font-medium hover:underline"
          >
            Learn more
            <svg
              className="w-4 h-4 ml-1"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2"
                d="M13 7l5 5m0 0l-5 5m5-5H6"
              ></path>
            </svg>
          </a>
        )}
      </div>
    </div>
  );
}

export default function Features() {
  return (
    <section id="features" className="py-16 md:py-24 mx-auto container px-4">
      <div className="text-left sm:text-center max-w-3xl mx-auto mb-16">
        <h2
          id="features-header"
          className="text-3xl md:text-5xl font-bold mb-6 bg-gradient-to-r from-gray-900 to-gray-700 dark:from-white dark:to-gray-300 bg-clip-text text-transparent"
        >
          Features Unique to Farseek
        </h2>
        <p className="text-xl text-gray-600 dark:text-gray-400">
          Powerful capabilities — built by the community to solve real-world
          challenges in infrastructure management
        </p>
      </div>

      <div className="space-y-8">
        <FeatureShowcase
          title="Stateless by Design"
          description="Forget about 'terraform.tfstate'. Farseek calculates infrastructure deltas directly from your cloud provider APIs and your Git history. No state files to corrupt, lock, or lose. Your infrastructure is always consistent with reality."
          version="1.0"
          align="left"
          color="bg-brand-500 text-white"
          codeExample={`# No backend configuration needed
# No remote state storage to manage
# Just standard HCL code

resource "google_storage_bucket" "website" {
  name          = "farseek-website"
  location      = "US"
  force_destroy = true

  website {
    main_page_suffix = "index.html"
    not_found_page   = "404.html"
  }
}`}
        />

        <FeatureShowcase
          title="Git IS the State"
          description="Your repository is the single source of truth. Farseek uses your Git commit history to determine resource lifecycles, tracking what was added, modified, or removed. If it's in main, it's deployed. If it's gone from Git, it's gone from the cloud."
          version="1.0"
          align="right"
          color="bg-brand-500 text-white"
          codeExample={`# Git log determines the plan
$ git log --oneline
a1b2c3d Remove unused database instance
e4f5g6h Add load balancer configuration
i7j8k9l Initial project setup

# Farseek sees the removal in a1b2c3d
# and destroys the resource automatically during apply.`}
        />

        <FeatureShowcase
          title="Zero-Lock Concurrency"
          description="Deploy unrelated stacks simultaneously. Since there's no state file to lock, your teams can work in parallel on different parts of the infrastructure without blocking each other. Scale your CI/CD pipelines without hitting state lock bottlenecks."
          version="1.0"
          align="left"
          color="bg-brand-500 text-white"
          codeExample={`# Team A deploys Frontend
$ cd frontend
$ farseek apply

# Team B deploys Backend at the same time
$ cd backend
$ farseek apply

# No Lock ID: 290834-239485... error.
# Just pure, parallel execution.`}
        />
      </div>
    </section>
  );
}
