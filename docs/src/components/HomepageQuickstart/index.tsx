import Translate from "@docusaurus/Translate";
import Tabs from "@theme/Tabs";
import TabItem from "@theme/TabItem";
import CodeBlock from "@theme/CodeBlock";
import styles from "./styles.module.css";

export default function HomepageQuickstart(): JSX.Element {
  return (
    <section className={styles.quickstart}>
      <div className="container">
        <div className={styles.sectionHeader}>
          <h2>
            <Translate id="quickstart.title">Quick Start</Translate>
          </h2>
          <p>
            <Translate id="quickstart.subtitle">
              Get up and running in under 5 minutes
            </Translate>
          </p>
        </div>
        <div className={styles.content}>
          <Tabs>
            <TabItem value="helm" label="Helm" default>
              <CodeBlock language="bash">
                {`# Add the KubeVision Helm repository
helm repo add kubevision https://kubevision.github.io/charts
helm repo update

# Install KubeVision
helm install kubevision gocronx/kubevision

# Temporarily access the dashboard from this machine
kubectl port-forward svc/kubevision 8080:8080
open http://localhost:8080`}
              </CodeBlock>
            </TabItem>
            <TabItem value="docker" label="Docker">
              <CodeBlock language="bash">
                {`# Build and run with Docker
docker build -f deploy/Dockerfile -t kubevision:latest .
docker run -p 8080:8080 \\
  -v ~/.kube/config:/root/.kube/config:ro \\
  kubevision:latest

# Open http://localhost:8080
# Default login: admin / admin123`}
              </CodeBlock>
            </TabItem>
            <TabItem value="dev" label="Development">
              <CodeBlock language="bash">
                {`git clone https://github.com/gocronx/kubevision.git
cd kubevision

# Backend — starts on :8080
go mod tidy && make dev

# Frontend — starts on :5173, proxies /api → :8080
cd web && pnpm install && pnpm dev`}
              </CodeBlock>
            </TabItem>
          </Tabs>
        </div>
      </div>
    </section>
  );
}
