import { themes as prismThemes } from "prism-react-renderer";
import type { Config } from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";

const config: Config = {
  title: "KubeVision",
  tagline: "See your clusters clearly, act on them instantly",
  favicon: "img/favicon.ico",

  url: "https://kubevision.github.io",
  baseUrl: "/",

  organizationName: "kubevision",
  projectName: "kubevision",

  onBrokenLinks: "throw",
  markdown: {
    hooks: {
      onBrokenMarkdownLinks: "warn",
    },
  },

  i18n: {
    defaultLocale: "en",
    locales: ["en", "zh-Hans"],
    localeConfigs: {
      en: { label: "English", direction: "ltr" },
      "zh-Hans": { label: "简体中文", direction: "ltr" },
    },
  },

  presets: [
    [
      "classic",
      {
        docs: {
          sidebarPath: "./sidebars.ts",
          editUrl:
            "https://github.com/gocronx/kubevision/tree/main/docs/",
        },
        blog: false,
        theme: {
          customCss: "./src/css/custom.css",
        },
      } satisfies Preset.Options,
    ],
  ],

  themes: [
    [
      "@easyops-cn/docusaurus-search-local",
      {
        hashed: true,
        language: ["en", "zh"],
        indexBlog: false,
      },
    ],
  ],

  themeConfig: {
    image: "img/kubevision-social.png",
    navbar: {
      title: "KubeVision",
      logo: {
        alt: "KubeVision Logo",
        src: "img/logo.svg",
      },
      items: [
        {
          type: "docSidebar",
          sidebarId: "docs",
          position: "left",
          label: "Docs",
        },
        {
          to: "/docs/api/overview",
          label: "API",
          position: "left",
        },
        {
          type: "localeDropdown",
          position: "right",
        },
        {
          href: "https://github.com/gocronx/kubevision",
          label: "GitHub",
          position: "right",
        },
      ],
    },
    footer: {
      style: "dark",
      links: [
        {
          title: "Docs",
          items: [
            { label: "Introduction", to: "/docs/intro" },
            { label: "Getting Started", to: "/docs/getting-started/installation" },
            { label: "Architecture", to: "/docs/architecture/overview" },
          ],
        },
        {
          title: "Community",
          items: [
            {
              label: "GitHub Discussions",
              href: "https://github.com/gocronx/kubevision/discussions",
            },
            {
              label: "Issues",
              href: "https://github.com/gocronx/kubevision/issues",
            },
          ],
        },
        {
          title: "More",
          items: [
            {
              label: "GitHub",
              href: "https://github.com/gocronx/kubevision",
            },
            { label: "Comparison", to: "/docs/comparison" },
            { label: "Roadmap", to: "/docs/roadmap" },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} KubeVision Contributors. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ["bash", "yaml", "go", "json"],
    },
    colorMode: {
      defaultMode: "light",
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
