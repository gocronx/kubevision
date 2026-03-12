import Translate from "@docusaurus/Translate";
import CodeBlock from "@theme/CodeBlock";
import styles from "./styles.module.css";

const architectureDiagram = `                    Browser
            ┌────────┴────────┐
            REST          WebSocket
            │                 │
┌───────────▼─────────────────▼──────────────┐
│  Middleware: RequestID → Logger → Auth       │
│                                             │
│  Handler ──→ Service ──→ K8sRepo            │
│                            │                │
│                 ┌──────────┴──────────┐     │
│                 │                     │     │
│          Informer Cache         API Server  │
│                 │                            │
│     Informer Watch ──→ EventListener        │
│                            │                │
│                        WS Hub ──→ Browser   │
│                                             │
│  DB: SQLite (dev) / PostgreSQL (prod)       │
└─────────────────────────────────────────────┘`;

export default function HomepageArchitecture(): JSX.Element {
  return (
    <section className={styles.architecture}>
      <div className="container">
        <div className={styles.sectionHeader}>
          <h2>
            <Translate id="architecture.title">Architecture</Translate>
          </h2>
          <p>
            <Translate id="architecture.subtitle">
              Clean layered design with real-time data flow
            </Translate>
          </p>
        </div>
        <div className={styles.twoCol}>
          <div className={styles.diagram}>
            <CodeBlock language="text">{architectureDiagram}</CodeBlock>
          </div>
          <div className={styles.description}>
            <div className={styles.flowItem}>
              <h3>
                📖 <Translate id="architecture.read.title">Read Path</Translate>
              </h3>
              <p>
                <Translate id="architecture.read.desc">
                  Informer cache first for sub-millisecond responses. Automatic
                  fallback to API Server when cache misses. 8 core resources
                  cached, 18+ on-demand.
                </Translate>
              </p>
            </div>
            <div className={styles.flowItem}>
              <h3>
                ✏️ <Translate id="architecture.write.title">Write Path</Translate>
              </h3>
              <p>
                <Translate id="architecture.write.desc">
                  API Server validates → Informer detects change → EventListener
                  triggers → WS Hub broadcasts → all browsers update instantly.
                </Translate>
              </p>
            </div>
            <div className={styles.flowItem}>
              <h3>
                🔌{" "}
                <Translate id="architecture.extend.title">
                  Extensibility
                </Translate>
              </h3>
              <p>
                <Translate id="architecture.extend.desc">
                  Add a new resource type with 1 line of Go (registry) + 1
                  config block (UI). CRDs auto-discovered at runtime. Plugin
                  system for Prometheus, Grafana, ArgoCD.
                </Translate>
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
