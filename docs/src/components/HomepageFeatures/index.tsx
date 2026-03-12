import Translate from "@docusaurus/Translate";
import styles from "./styles.module.css";

interface Feature {
  icon: string;
  titleId: string;
  titleDefault: string;
  descId: string;
  descDefault: string;
}

const features: Feature[] = [
  {
    icon: "⚡",
    titleId: "features.realtime.title",
    titleDefault: "Real-time Sync",
    descId: "features.realtime.desc",
    descDefault:
      "Informer Watch → WebSocket Push. Sub-second updates across all connected clients, zero polling overhead.",
  },
  {
    icon: "🌐",
    titleId: "features.multicluster.title",
    titleDefault: "Multi-cluster",
    descId: "features.multicluster.desc",
    descDefault:
      "Manage all your clusters from a single dashboard. Switch contexts instantly, compare resources across environments.",
  },
  {
    icon: "💻",
    titleId: "features.terminal.title",
    titleDefault: "Terminal & Logs",
    descId: "features.terminal.desc",
    descDefault:
      "Full Pod terminal with session recording & playback. Real-time log streaming with search and filtering.",
  },
  {
    icon: "🔒",
    titleId: "features.security.title",
    titleDefault: "Enterprise Security",
    descId: "features.security.desc",
    descDefault:
      "5-level RBAC, TOTP 2FA with recovery codes, audit logging, Secrets masking, and instant token revocation.",
  },
  {
    icon: "🗺️",
    titleId: "features.topology.title",
    titleDefault: "Resource Topology",
    descId: "features.topology.desc",
    descDefault:
      "Visual ownership graph showing relationships between Deployments, ReplicaSets, Pods, Services, and more.",
  },
  {
    icon: "🔍",
    titleId: "features.search.title",
    titleDefault: "Global Search",
    descId: "features.search.desc",
    descDefault:
      "Cmd+K fuzzy search across all clusters, namespaces, and resource types. Find anything in milliseconds.",
  },
];

export default function HomepageFeatures(): JSX.Element {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className={styles.sectionHeader}>
          <h2>
            <Translate id="features.title">Core Features</Translate>
          </h2>
          <p>
            <Translate id="features.subtitle">
              Everything you need to manage Kubernetes clusters at scale
            </Translate>
          </p>
        </div>
        <div className={styles.grid}>
          {features.map((feature, idx) => (
            <div key={idx} className={styles.card}>
              <div className={styles.cardIcon}>{feature.icon}</div>
              <h3>
                <Translate id={feature.titleId}>
                  {feature.titleDefault}
                </Translate>
              </h3>
              <p>
                <Translate id={feature.descId}>{feature.descDefault}</Translate>
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
