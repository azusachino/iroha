export const site = {
  name: "harus tracks",
  byline: "by iroha",
  version: import.meta.env.VITE_IROHA_VERSION ?? "dev",
  description:
    "A public, privacy-trimmed window into a personal activity archive.",
  repositoryUrl: "https://github.com/azusachino/iroha",
} as const;
