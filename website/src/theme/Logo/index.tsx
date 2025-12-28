import React from "react";
import Link from "@docusaurus/Link";
import useBaseUrl from "@docusaurus/useBaseUrl";
import ThemedImage from "@theme/ThemedImage";

export default function Logo() {
  const logoLink = useBaseUrl("/");
  const sources = {
    light: useBaseUrl("/img/farseek_white.png"),
    dark: useBaseUrl("/img/farseek_black.png"),
  };

  return (
    <Link
      to={logoLink}
      className="flex items-center"
      aria-label="Go to homepage"
    >
      <ThemedImage
        alt="Farseek Logo"
        sources={sources}
        className="h-9 mb-2 sm:h-12 sm:mb-3 hover:text-brand-500 dark:hover:text-brand-500"
      />
    </Link>
  );
}
