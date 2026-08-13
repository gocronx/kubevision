import useDocusaurusContext from "@docusaurus/useDocusaurusContext";
import Layout from "@theme/Layout";
import HomepageHero from "../components/HomepageHero";
import HomepageFeatures from "../components/HomepageFeatures";
import HomepageArchitecture from "../components/HomepageArchitecture";
import HomepageUniqueCapabilities from "../components/HomepageUniqueCapabilities";
import HomepageQuickstart from "../components/HomepageQuickstart";

export default function Home(): JSX.Element {
  const { siteConfig } = useDocusaurusContext();
  return (
    <Layout
      title={siteConfig.title}
      description="A modern, real-time Kubernetes multi-cluster dashboard with 2FA, audit logging, and dry-run diff"
    >
      <HomepageHero />
      <main>
        <HomepageFeatures />
        <HomepageArchitecture />
        <HomepageUniqueCapabilities />
        <HomepageQuickstart />
      </main>
    </Layout>
  );
}
