import { getAccessToken } from '$lib/stores/auth.svelte'
import { failUnauthorized, refreshTokens } from '$lib/api/client'

export type StreamEvent = {
  type: string
  task_id: number
  comment_id: number
}

type EventCallback = (event: StreamEvent) => void

const ENDPOINT = '/api/v1/events/stream'

const BASE_RETRY_MS = 1000
const MAX_RETRY_MS = 30000

const subscribers = new Set<EventCallback>()
let activeController: AbortController | null = null

/**
 * Subscribe to the live events stream. Returns an unsubscribe function.
 * Subscribers are independent of the connection lifecycle: events are only
 * delivered while a connection is open (see startEventStream).
 */
export function onStreamEvent(callback: EventCallback): () => void {
  subscribers.add(callback)
  return () => {
    subscribers.delete(callback)
  }
}

/**
 * Open (or re-open) the events stream and keep it connected with backoff.
 * Returns a stop function that aborts the connection and its reconnect loop.
 * Safe to call again: any previously started stream is stopped first.
 */
export function startEventStream(): () => void {
  activeController?.abort()

  const controller = new AbortController()
  activeController = controller
  void runEventLoop(controller)

  return () => {
    controller.abort()
    if (activeController === controller) activeController = null
  }
}

function dispatch(event: StreamEvent): void {
  for (const callback of subscribers) {
    try {
      callback(event)
    } catch (err) {
      // A failing subscriber must not tear down the shared stream.
      console.error('events stream subscriber failed', err)
    }
  }
}

function parseFrame(lines: string[]): StreamEvent | null {
  let type = ''
  const dataLines: string[] = []

  for (const line of lines) {
    if (line.startsWith(':')) continue
    if (line.startsWith('event:')) type = line.slice(6).trim()
    else if (line.startsWith('data:')) dataLines.push(line.slice(5).trim())
  }

  if (dataLines.length === 0) return null

  try {
    const data = JSON.parse(dataLines.join('\n'))
    return {
      type: type || String(data.type ?? ''),
      task_id: Number(data.task_id ?? 0),
      comment_id: Number(data.comment_id ?? 0),
    }
  } catch {
    return null
  }
}

async function runEventLoop(controller: AbortController): Promise<void> {
  const signal = controller.signal
  let attempt = 0

  while (!signal.aborted) {
    try {
      await runAttempt(controller)
      // A stream that ended cleanly should be re-established promptly.
      attempt = 0
    } catch {
      if (signal.aborted) return
    }

    if (signal.aborted) return

    const delay = Math.min(BASE_RETRY_MS * 2 ** attempt, MAX_RETRY_MS)
    attempt++
    await sleep(delay, signal)
  }
}

async function runAttempt(controller: AbortController): Promise<void> {
  const signal = controller.signal

  const token = getAccessToken()
  if (!token) {
    // Session gone (or not yet restored); wait for the owner to authenticate.
    await sleep(BASE_RETRY_MS, signal)
    return
  }

  let res = await fetch(ENDPOINT, {
    headers: { Authorization: `Bearer ${token}` },
    signal,
  })

  if (res.status === 401) {
    const retryToken = (await refreshTokens()) ? getAccessToken() : null
    if (!retryToken) {
      failUnauthorized()
      controller.abort()
      return
    }
    res = await fetch(ENDPOINT, {
      headers: { Authorization: `Bearer ${retryToken}` },
      signal,
    })
  }

  if (!res.ok || !res.body) {
    throw new Error(`events stream connect failed with status ${res.status}`)
  }

  await readEventStream(res.body, signal)
}

async function readEventStream(
  body: ReadableStream<Uint8Array>,
  signal: AbortSignal,
): Promise<void> {
  const reader = body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let frameLines: string[] = []

  try {
    while (!signal.aborted) {
      const { done, value } = await reader.read()
      if (done) return

      buffer += decoder.decode(value, { stream: true })

      let newline = buffer.indexOf('\n')
      while (newline >= 0) {
        let line = buffer.slice(0, newline)
        buffer = buffer.slice(newline + 1)
        if (line.endsWith('\r')) line = line.slice(0, -1)

        if (line === '') {
          if (frameLines.length > 0) {
            const event = parseFrame(frameLines)
            frameLines = []
            if (event) dispatch(event)
          }
        } else {
          frameLines.push(line)
        }

        newline = buffer.indexOf('\n')
      }
    }
  } finally {
    void reader.cancel().catch(() => {})
  }
}

function sleep(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    if (signal.aborted) {
      resolve()
      return
    }

    const onAbort = () => {
      clearTimeout(timer)
      resolve()
    }
    const timer = setTimeout(() => {
      signal.removeEventListener('abort', onAbort)
      resolve()
    }, ms)
    signal.addEventListener('abort', onAbort, { once: true })
  })
}
