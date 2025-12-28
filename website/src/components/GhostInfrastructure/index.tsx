import React from "react";
import Link from "@docusaurus/Link";

export default function GhostInfrastructure() {
  return (
    <section className="py-24 bg-gray-50 dark:bg-gray-900 overflow-hidden">
      <div className="container mx-auto px-4">
        <div className="max-w-4xl mx-auto text-center mb-16">
          <h2 className="text-3xl md:text-5xl font-bold mb-6 text-gray-900 dark:text-white">
            The Ghost Infrastructure Problem
          </h2>
          <p className="text-xl text-gray-600 dark:text-gray-400">
            Agentic development is making infrastructure creation instant. <br className="hidden md:block" />
            But without traceability, it's creating a crisis.
          </p>
        </div>

        <div className="grid md:grid-cols-2 gap-8 items-stretch">
          {/* Problem Card */}
          <div className="bg-white dark:bg-gray-800 rounded-2xl p-8 border border-red-100 dark:border-red-900/30 shadow-lg relative overflow-hidden group hover:-translate-y-1 transition-transform duration-300">
            <div className="absolute top-0 right-0 p-4 opacity-10">
              <svg className="w-32 h-32 text-red-500" fill="currentColor" viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>
            </div>
            <h3 className="text-2xl font-bold mb-4 text-red-600 dark:text-red-400 flex items-center gap-2">
              <span className="text-3xl">👻</span> The Status Quo
            </h3>
            <ul className="space-y-4 text-lg text-gray-700 dark:text-gray-300">
              <li className="flex items-start gap-3">
                <span className="text-red-500 text-xl font-bold">×</span>
                <span>AI agents spin up resources without records</span>
              </li>
              <li className="flex items-start gap-3">
                <span className="text-red-500 text-xl font-bold">×</span>
                <span>Infrastructure exists in Cloud, but not in Git</span>
              </li>
              <li className="flex items-start gap-3">
                <span className="text-red-500 text-xl font-bold">×</span>
                <span>"Ghost resources" bleed costs for months</span>
              </li>
              <li className="flex items-start gap-3">
                <span className="text-red-500 text-xl font-bold">×</span>
                <span>No audit trail for compliance or security</span>
              </li>
            </ul>
          </div>

          {/* Solution Card */}
          <div className="bg-brand-50 dark:bg-brand-900/10 rounded-2xl p-8 border border-brand-100 dark:border-brand-900/30 shadow-lg relative overflow-hidden group hover:-translate-y-1 transition-transform duration-300">
             <div className="absolute top-0 right-0 p-4 opacity-10">
              <svg className="w-32 h-32 text-brand-500" fill="currentColor" viewBox="0 0 24 24"><path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/></svg>
            </div>
            <h3 className="text-2xl font-bold mb-4 text-brand-600 dark:text-brand-400 flex items-center gap-2">
              <span className="text-3xl">🛡️</span> The Farseek Way
            </h3>
            <ul className="space-y-4 text-lg text-gray-700 dark:text-gray-300">
              <li className="flex items-start gap-3">
                <span className="text-brand-500 text-xl font-bold">✓</span>
                <span>Agents commit .tf files to create resources</span>
              </li>
              <li className="flex items-start gap-3">
                <span className="text-brand-500 text-xl font-bold">✓</span>
                <span>Run <code>farseek apply</code> to sync Git state instantly</span>
              </li>
              <li className="flex items-start gap-3">
                <span className="text-brand-500 text-xl font-bold">✓</span>
                <span>Every dollar traced back to a commit</span>
              </li>
              <li className="flex items-start gap-3">
                <span className="text-brand-500 text-xl font-bold">✓</span>
                <span>Traceability is the <i>fast</i> path</span>
              </li>
            </ul>
          </div>
        </div>

        <div className="mt-12 text-center">
          <Link
            className="inline-flex items-center gap-2 text-lg font-semibold text-brand-600 dark:text-brand-400 hover:text-brand-800 dark:hover:text-brand-300 transition-colors"
            to="/blog/the-coming-crisis-of-untraceable-infrastructure">
            Read the full manifesto
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17 8l4 4m0 0l-4 4m4-4H3"></path></svg>
          </Link>
        </div>
      </div>
    </section>
  );
}
