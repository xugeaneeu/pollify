let counter = 0

export function uniqueEmail(prefix = 'user'): string {
  counter += 1
  const stamp = `${Date.now().toString(36)}-${counter}`
  return `${prefix}.${stamp}@example.com`
}

function pad(n: number): string {
  return n.toString().padStart(2, '0')
}

function toLocalDatetimeInputValue(d: Date): string {
  return (
    d.getFullYear() +
    '-' +
    pad(d.getMonth() + 1) +
    '-' +
    pad(d.getDate()) +
    'T' +
    pad(d.getHours()) +
    ':' +
    pad(d.getMinutes())
  )
}

export function pastLocal(minutesAgo = 60): string {
  const d = new Date(Date.now() - minutesAgo * 60_000)
  return toLocalDatetimeInputValue(d)
}

export function futureLocal(minutesAhead = 60 * 24): string {
  const d = new Date(Date.now() + minutesAhead * 60_000)
  return toLocalDatetimeInputValue(d)
}
