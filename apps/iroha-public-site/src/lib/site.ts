export const site = {
  name: "harus track",
  byline: "",
  version: import.meta.env.VITE_IROHA_VERSION ?? "dev",
  description:
    "A public, privacy-trimmed window into a personal activity archive.",
  repositoryUrl: "https://github.com/azusachino/iroha",
} as const;
