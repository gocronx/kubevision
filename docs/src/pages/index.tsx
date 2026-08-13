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
      description="An AI-native Kubernetes dashboard for context-aware troubleshooting and RBAC-controlled cluster operations"
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
