import styles from './App.module.css'
import { ServerStatus } from './features/server/ServerStatus'

function App() {
  return (
    <main className={styles.page}>
      <h1 className={styles.title}>Amethyst</h1>
      <p className={styles.tagline}>
        Community discussions across independently operated servers.
      </p>
      <ServerStatus />
    </main>
  )
}

export default App
