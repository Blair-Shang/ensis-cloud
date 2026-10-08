import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { test } from 'node:test'
import { fileURLToPath } from 'node:url'

import { planRelease, promote, publishedSet } from './cut-changelog.mjs'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const script = path.join(root, '.github/scripts/cut-changelog.mjs')

const sample = `# 变更日志

说明。

## [Unreleased]

### 修复

- 流水线按 changelog 自动发版。

## [1.2.1] - 2026-09-22

### 变更

- 已有条目。
`

test('plans the package version when Unreleased has list items', () => {
  assert.equal(planRelease(sample, publishedSet('1.2.1'), '1.2.2'), '1.2.2')
})

test('plans an existing version section that is not tagged yet', () => {
  const text = `## [Unreleased]\n\n## [1.2.2] - 2026-10-08\n\n- 已写好。\n`
  assert.equal(planRelease(text, publishedSet('1.2.1'), '1.2.2'), '1.2.2')
})

test('skips a version that already has a tag', () => {
  assert.equal(planRelease(sample, publishedSet('v1.2.2,1.2.1'), '1.2.2'), '')
})

test('skips when there is no list item', () => {
  const text = `## [Unreleased]\n\n### 修复\n\n## [1.2.1] - 2026-09-22\n\n- 旧条目。\n`
  assert.equal(planRelease(text, publishedSet('1.2.1'), '1.2.2'), '')
})

test('rejects a non semver package version', () => {
  assert.throws(() => planRelease(sample, publishedSet(''), '1.2'), /x\.y\.z/)
})

test('moves Unreleased list items into a dated version section', () => {
  const next = promote(sample, '1.2.2', '2026-10-08')
  assert.equal(
    next,
    `# 变更日志

说明。

## [Unreleased]

## [1.2.2] - 2026-10-08

### 修复

- 流水线按 changelog 自动发版。

## [1.2.1] - 2026-09-22

### 变更

- 已有条目。
`,
  )
  assert.equal(promote(next, '1.2.2', '2026-10-09'), next)
})

test('fills an empty version section and dates an undated heading', () => {
  const text = `## [Unreleased]\n\n- 新条目。\n\n## [1.2.2]\n\n## [1.2.1] - 2026-09-22\n\n- 旧条目。\n`
  assert.equal(
    promote(text, '1.2.2', '2026-10-08'),
    `## [Unreleased]\n\n## [1.2.2] - 2026-10-08\n\n- 新条目。\n\n## [1.2.1] - 2026-09-22\n\n- 旧条目。\n`,
  )
})

test('keeps Unreleased notes when the version section already has its own items', () => {
  assert.equal(promote(sample, '1.2.1', '2026-10-08'), sample)
})

test('leaves the checked-in changelog untouched for the current tagged section', () => {
  const original = fs.readFileSync(path.join(root, 'CHANGELOG.md'), 'utf8')
  assert.equal(promote(original, '1.2.1', '2026-10-08'), original)
  assert.equal(planRelease(original, publishedSet('1.2.1'), '1.2.1'), '')
})

test('plan command prints only the version', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'cut-changelog-'))
  const changelog = path.join(dir, 'CHANGELOG.md')
  fs.writeFileSync(changelog, sample, 'utf8')
  const result = spawnSync(
    process.execPath,
    [script, 'plan', '--version', '1.2.2', '--published', '1.2.1', '--date', '2026-10-08', '--changelog', changelog],
    { encoding: 'utf8' },
  )
  assert.equal(result.status, 0)
  assert.equal(result.stdout, '1.2.2\n')
  assert.equal(fs.readFileSync(changelog, 'utf8'), sample)

  const bad = spawnSync(process.execPath, [script, 'plan', '--version', '1.2', '--published', '', '--date', '2026-10-08', '--changelog', changelog], {
    encoding: 'utf8',
  })
  assert.equal(bad.status, 2)
})

test('apply command writes the version section', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'cut-changelog-'))
  const changelog = path.join(dir, 'CHANGELOG.md')
  fs.writeFileSync(changelog, sample, 'utf8')
  const result = spawnSync(
    process.execPath,
    [script, 'apply', '--version', '1.2.2', '--date', '2026-10-08', '--changelog', changelog],
    { encoding: 'utf8' },
  )
  assert.equal(result.status, 0)
  assert.equal(result.stdout, '1.2.2\n')
  assert.match(fs.readFileSync(changelog, 'utf8'), /## \[1\.2\.2\] - 2026-10-08\n\n### 修复/)
})
