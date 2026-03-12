import Translate from "@docusaurus/Translate";
import Link from "@docusaurus/Link";
import styles from "./styles.module.css";

const rows = [
  { feature: "Multi-cluster", kv: true, headlamp: true, k9s: true, kuboard: true },
  { feature: "2FA (TOTP)", kv: true, headlamp: false, k9s: false, kuboard: false },
  { feature: "Dry-Run Diff", kv: true, headlamp: false, k9s: false, kuboard: false },
  { feature: "Cross-cluster Diff", kv: true, headlamp: false, k9s: false, kuboard: false },
  { feature: "Terminal Recording", kv: true, headlamp: false, k9s: false, kuboard: false },
  { feature: "Secrets Masking", kv: true, headlamp: false, k9s: false, kuboard: false },
  { feature: "kubectl Hints", kv: true, headlamp: false, k9s: false, kuboard: false },
  { feature: "Audit Logging", kv: true, headlamp: false, k9s: false, kuboard: true },
  { feature: "Resource Topology", kv: true, headlamp: true, k9s: true, kuboard: true },
  { feature: "Real-time WebSocket", kv: true, headlamp: true, k9s: true, kuboard: true },
  { feature: "Global Search", kv: true, headlamp: true, k9s: true, kuboard: false },
  { feature: "Plugin System", kv: true, headlamp: true, k9s: true, kuboard: false },
  { feature: "Dark Mode", kv: true, headlamp: true, k9s: true, kuboard: false },
  { feature: "i18n", kv: true, headlamp: true, k9s: false, kuboard: true },
];

function Cell({ value }: { value: boolean }) {
  return (
    <td className={value ? styles.yes : styles.no}>{value ? "✓" : "—"}</td>
  );
}

export default function HomepageComparison(): JSX.Element {
  return (
    <section className={styles.comparison}>
      <div className="container">
        <div className={styles.sectionHeader}>
          <h2>
            <Translate id="comparison.title">How We Compare</Translate>
          </h2>
          <p>
            <Translate id="comparison.subtitle">
              KubeVision vs popular Kubernetes dashboards
            </Translate>
          </p>
        </div>
        <div className={styles.tableWrapper}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>
                  <Translate id="comparison.feature">Feature</Translate>
                </th>
                <th className={styles.highlight}>KubeVision</th>
                <th>Headlamp</th>
                <th>K9s</th>
                <th>Kuboard</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row, idx) => (
                <tr key={idx}>
                  <td>{row.feature}</td>
                  <Cell value={row.kv} />
                  <Cell value={row.headlamp} />
                  <Cell value={row.k9s} />
                  <Cell value={row.kuboard} />
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className={styles.cta}>
          <Link to="/docs/comparison" className={styles.ctaLink}>
            <Translate id="comparison.cta">
              View full comparison →
            </Translate>
          </Link>
        </div>
      </div>
    </section>
  );
}
