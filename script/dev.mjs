/**
 * 同时启动 Cloud Go API 与 Admin Vite。Ctrl+C / SIGTERM 会结束整组进程。
 */
import { spawn } from 'node:child_process'

const isWin = process.platform === 'win32'

/** @type {import('node:child_process').ChildProcess[]} */
const children = []
let exiting = false

/**
 * 拉起一条前台命令，stdio 直通当前终端。
 * @param {string} label
 * @param {string} command
 * @param {string[]} args
 */
function run(label, command, args) {
  console.log(`[dev] start ${label}: ${command} ${args.join(' ')}`)
  const child = spawn(command, args, {
    stdio: 'inherit',
    shell: isWin,
    env: process.env,
    cwd: process.cwd(),
  })
  child.on('exit', (code, signal) => {
    if (exiting) {
      return
    }
    const reason = code != null ? `code ${code}` : String(signal)
    console.log(`[dev] ${label} exited (${reason})`)
  })
  children.push(child)
}

function shutdown() {
  if (exiting) {
    return
  }
  exiting = true
  const pending = children.filter((child) => child.pid && child.exitCode == null)
  if (pending.length === 0) {
    process.exit(0)
    return
  }
  let left = pending.length
  for (const child of pending) {
    child.once('exit', () => {
      left -= 1
      if (left === 0) {
        process.exit(0)
      }
    })
    if (isWin) {
      spawn('taskkill', ['/pid', String(child.pid), '/T', '/F'], {
        stdio: 'ignore',
        windowsHide: true,
      })
    } else {
      child.kill('SIGTERM')
    }
  }
  setTimeout(() => process.exit(0), 4000)
}

process.on('SIGINT', shutdown)
process.on('SIGTERM', shutdown)

run('server', 'go', ['run', '-C', 'server', './cmd/server'])
run('admin', 'pnpm', ['exec', 'vite', '--config', 'admin/vite.config.ts'])
