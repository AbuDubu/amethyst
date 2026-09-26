import { useServerInfo } from './useServerInfo'
import styles from './ServerStatus.module.css'

/** A small line identifying the running server, or why it can't be reached. */
export function ServerStatus() {
  const { data, isPending, isError } = useServerInfo()

  if (isPending) {
    return <p className={styles.status}>Checking server…</p>
  }
  if (isError) {
    return (
      <p className={styles.status} role="alert">
        Couldn’t reach the server. Try again shortly.
      </p>
    )
  }
  return (
    <p className={styles.status}>
      {data.canonical_origin} · running {data.software} {data.version}
    </p>
  )
}
