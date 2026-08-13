import clsx from "clsx";
import Link from "@docusaurus/Link";
import Translate from "@docusaurus/Translate";
import styles from "./styles.module.css";

export default function HomepageHero(): JSX.Element {
  return (
    <header className={styles.hero}>
      <div className="container">
        <div className={styles.heroInner}>
          <h1 className={styles.heroTitle}>
            <span className={styles.heroLogo}>⎈</span>
            KubeVision
          </h1>
          <p className={styles.heroSubtitle}>
            <Translate id="hero.tagline">
              AI-native Kubernetes operations with human control
            </Translate>
          </p>
          <p className={styles.heroDescription}>
            <Translate id="hero.description">
              Investigate live cluster state, understand failures, and execute
              RBAC-controlled actions through natural language. Every mutation
              requires confirmation and remains auditable.
            </Translate>
          </p>
          <div className={styles.heroButtons}>
            <Link
              className={clsx("button button--primary button--lg", styles.heroBtn)}
              to="/docs/getting-started/installation"
            >
              <Translate id="hero.getStarted">Get Started</Translate>
            </Link>
            <Link
              className={clsx(
                "button button--outline button--lg",
                styles.heroBtn,
                styles.heroBtnSecondary
              )}
              href="https://github.com/gocronx/kubevision"
            >
              GitHub →
            </Link>
          </div>
        </div>
      </div>
    </header>
  );
}
