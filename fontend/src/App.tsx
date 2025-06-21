import { useState } from 'react'
import { createPromiseClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { GreetService } from './gen/proto/fawa/greet/v1/hello_connect'
import './App.css'

// The transport defines how we connect to the server.
const transport = createConnectTransport({
  baseUrl: 'http://localhost:8081',
})

// Here we make the client itself, combining the service
// definition with the transport.
const client = createPromiseClient(GreetService, transport)

function App() {
  const [name, setName] = useState('World')
  const [logs, setLogs] = useState<string[]>([])

  const addLog = (message: string) => {
    setLogs((prev) => [...prev, message])
  }

  const clearLogs = () => {
    setLogs([])
  }

  const runSayHello = async () => {
    clearLogs()
    addLog('--- Unary RPC ---')
    addLog(`[${new Date().toLocaleTimeString()}] Sending: ${name}`)
    const response = await client.sayHello({ name })
    addLog(`[${new Date().toLocaleTimeString()}] Received: ${response.resp}`)
    addLog('--- Finished ---')
  }

  const runGreetStream = async () => {
    clearLogs()
    addLog('--- Server-Streaming RPC ---')
    addLog(`[${new Date().toLocaleTimeString()}] Sending request...`)
    try {
      for await (const response of client.greetStream({ name })) {
        addLog(`[${new Date().toLocaleTimeString()}] Received stream part: ${response.part}`)
      }
      addLog(`[${new Date().toLocaleTimeString()}] Server stream closed.`)
    } catch (e) {
      addLog(`[${new Date().toLocaleTimeString()}] Error: ${e instanceof Error ? e.message : 'Unknown error'}`)
    }
    addLog('--- Finished ---')
  }

  const runGreetClientStream = async () => {
    clearLogs()
    addLog('--- Client-Streaming RPC ---')
    const names = ['Alice', 'Bob', 'Charlie']
    async function* sendNames() {
      for (const n of names) {
        addLog(`[${new Date().toLocaleTimeString()}] Sending stream part: ${n}`)
        yield { name: n }
        await new Promise((resolve) => setTimeout(resolve, 500))
      }
    }
    addLog(`[${new Date().toLocaleTimeString()}] Client stream started...`)
    const response = await client.greetClientStream(sendNames())
    addLog(`[${new Date().toLocaleTimeString()}] Server replied with summary: ${response.summary}`)
    addLog('--- Finished ---')
  }

  const runGreetBidiStream = async () => {
    clearLogs()
    addLog('--- Bidirectional-Streaming RPC ---')
    const names = ['ECHO-1', 'ECHO-2', 'ECHO-3']
    async function* sendNames() {
      for (const n of names) {
        addLog(`[${new Date().toLocaleTimeString()}] Sending: ${n}`)
        yield { name: n }
        await new Promise((resolve) => setTimeout(resolve, 500))
      }
    }
    addLog(`[${new Date().toLocaleTimeString()}] Bidi stream started...`)
    try {
      const stream = client.greetBidiStream(sendNames())
      for await (const response of stream) {
        addLog(`[${new Date().toLocaleTimeString()}] Received: ${response.echo}`)
      }
      addLog(`[${new Date().toLocaleTimeString()}] Server stream closed.`)
    } catch (e) {
      addLog(`[${new Date().toLocaleTimeString()}] Error: ${e instanceof Error ? e.message : 'Unknown error'}`)
    }
    addLog('--- Finished ---')
  }

  return (
    <>
      <h1>fawa Greet Service</h1>
      <div className="card">
        <label>
          Name:
          <input type="text" value={name} onChange={(e) => setName(e.target.value)} />
        </label>
      </div>
      <div className="card">
        <button onClick={runSayHello}>SayHello (Unary)</button>
        <button onClick={runGreetStream}>GreetStream (Server Stream)</button>
        <button onClick={runGreetClientStream}>GreetClientStream (Client Stream)</button>
        <button onClick={runGreetBidiStream}>GreetBidiStream (Bidi Stream)</button>
      </div>
      <div className="card">
        <h3>Logs</h3>
        <button onClick={clearLogs} style={{ float: 'right' }}>Clear</button>
        <pre className="logs">
          {logs.map((log, i) => (
            <div key={i}>{log}</div>
          ))}
        </pre>
      </div>
    </>
  )
}

export default App
