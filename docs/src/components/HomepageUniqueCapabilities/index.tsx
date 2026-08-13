import Translate from "@docusaurus/Translate";
import styles from "./styles.module.css";

interface Capability {
  icon: string;
  titleId: string;
  titleDefault: string;
  descId: string;
  descDefault: string;
}

const capabilities: Capability[] = [
  {
    icon: "🔐",
    titleId: "unique.2fa.title",
    titleDefault: "2FA (TOTP)",
    descId: "unique.2fa.desc",
    descDefault:
      "Two-factor authentication with QR setup, recovery codes, and admin-enforced enrollment. No other K8s dashboard offers this.",
  },
  {
    icon: "🔬",
    titleId: "unique.dryrun.title",
    titleDefault: "Dry-Run Diff",
    descId: "unique.dryrun.desc",
    descDefault:
      "Preview every change before applying. API Server validates your YAML and shows a side-by-side diff.",
  },
  {
    icon: "🎬",
    titleId: "unique.recording.title",
    titleDefault: "Terminal Recording",
    descId: "unique.recording.desc",
    descDefault:
      "Every Pod/Node terminal session recorded in asciinema format. Admins can replay sessions for audit and training.",
  },
  {
    icon: "⌨️",
    titleId: "unique.kubectl.title",
    titleDefault: "kubectl Hints",
    descId: "unique.kubectl.desc",
    descDefault:
      "Every UI action shows the equivalent kubectl command. Learn Kubernetes CLI while using the dashboard.",
  },
  {
    icon: "🙈",
    titleId: "unique.secrets.title",
    titleDefault: "Secrets Masking",
    descId: "unique.secrets.desc",
    descDefault:
      "Secrets are masked by default in all views. Explicit action required to reveal — no accidental exposure.",
  },
];

export default function HomepageUniqueCapabilities(): JSX.Element {
  return (
    <section className={styles.unique}>
      <div className="container">
        <div className={styles.sectionHeader}>
          <span className={styles.badge}>
            <Translate id="unique.badge">Only in KubeVision</Translate>
          </span>
          <h2>
            <Translate id="unique.title">Unique Capabilities</Translate>
          </h2>
          <p>
            <Translate id="unique.subtitle">
              Features no other Kubernetes dashboard provides
            </Translate>
          </p>
        </div>
        <div className={styles.grid}>
          {capabilities.map((cap, idx) => (
            <div key={idx} className={styles.card}>
              <div className={styles.cardIcon}>{cap.icon}</div>
              <h3>
                <Translate id={cap.titleId}>{cap.titleDefault}</Translate>
              </h3>
              <p>
                <Translate id={cap.descId}>{cap.descDefault}</Translate>
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
