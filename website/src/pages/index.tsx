import React from "react";
import Layout from "@theme/Layout";
import Hero from "../components/Hero";
import Features from "../components/Features";

import GhostInfrastructure from "../components/GhostInfrastructure";
import FAQ from "../components/FAQ";
import HowToContribute from "../components/HowToContribute";

export default function Home() {
  return (
    <Layout description="Farseek - Stateless Infrastructure as Code">
      <Hero />
      <GhostInfrastructure />
      <Features />
      <HowToContribute />
      <FAQ />

    </Layout>
  );
}
