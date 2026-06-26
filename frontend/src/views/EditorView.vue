<template>
  <div class="editor-layout">
    <aside class="editor-sidebar">
      <div class="sidebar-hd">
        <h3>Scripts</h3>
        <button class="btn btn-info btn-sm" @click="newScript">＋ New</button>
      </div>
      <ul class="sidebar-list">
        <li v-for="s in scripts" :key="s.id" :class="{ active: active?.id === s.id }" @click="selectScript(s)">
          <div>
            <div class="si-name">{{ s.label || s.name }}</div>
            <div class="si-meta">{{ s.name }}{{ s.type ? ' · ' + s.type : '' }}</div>
          </div>
          <button class="si-del" @click.stop="delScript(s)">×</button>
        </li>
      </ul>
    </aside>
    <main class="editor-main">
      <div class="editor-toolbar" v-if="active">
        <input v-model="name" class="name" placeholder="name (unique)" />
        <input v-model="label" class="label" placeholder="display name" />
        <input v-model="type" class="type-field" placeholder="type" />
        <div class="spacer"></div>
        <button class="btn btn-warning btn-sm" @click="debug">● Debug</button>
        <button class="btn btn-success btn-sm" @click="handleSave">Save</button>
        <button class="btn btn-info btn-sm" @click="handleRun">▸ Run</button>
      </div>

      <div v-if="active" class="cm-wrap" ref="editorHost"></div>

      <div v-if="!active" class="welcome-state">
        <div class="welcome-icon">⚡</div>
        <h2>Tiny Lowcode Editor</h2>
        <p>Select a script from the sidebar or create a new one.</p>
        <div class="welcome-shortcuts">
          <div class="shortcut"><kbd>⌘/Ctrl</kbd> + <kbd>S</kbd> Save</div>
          <div class="shortcut"><kbd>⌘/Ctrl</kbd> + <kbd>Enter</kbd> Run</div>
        </div>
      </div>

      <div class="editor-output">
        <div class="output-tabs">
          <span :class="{ active: outTab==='output' }" @click="outTab='output'">Output</span>
          <span :class="{ active: outTab==='debug' }" @click="outTab='debug'">Debug</span>
          <span style="margin-left:auto;color:var(--text-muted)">{{ rowCount }}</span>
        </div>
        <div v-if="outTab==='output'" class="output-content" :class="{ error: hasError }" v-text="output"></div>
        <div v-else class="output-content" v-html="debugHtml"></div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick, shallowRef } from 'vue'
import api from '../utils/api'
import { EditorState } from '@codemirror/state'
import { EditorView, keymap, lineNumbers, gutter, GutterMarker } from '@codemirror/view'
import { javascript } from '@codemirror/lang-javascript'
import { oneDark } from '@codemirror/theme-one-dark'
import { defaultKeymap, history, indentWithTab } from '@codemirror/commands'

const scripts = ref([])
const active = ref(null)
const name = ref('')
const label = ref('')
const type = ref('')
const output = ref('Run a script to see output')
const hasError = ref(false)
const outTab = ref('output')
const rowCount = ref('')
const debugHtml = ref('')
const editorHost = ref(null)
const cmView = shallowRef(null)
let breakpoints = new Set()

onMounted(loadScripts)

async function loadScripts() {
  try {
    const res = await api.get('/api/scripts')
    scripts.value = Array.isArray(res.data) ? res.data : []
  } catch (e) {
    console.error('loadScripts failed:', e)
  }
}

function newScript() {
  active.value = { id: null, name: '', label: 'Untitled', type: '', tsCode: '' }
  name.value = ''; label.value = 'Untitled'; type.value = ''
  nextTick(() => initEditor(''))
}

async function selectScript(s) {
  try {
    const res = await api.get(`/api/scripts/${s.id}`)
    const data = res.data
    active.value = data
    name.value = data.name || ''
    label.value = data.label || ''
    type.value = data.type || ''
    await nextTick()
    initEditor(data.tsCode || '')
    loadBreakpoints()
  } catch (e) {
    console.error('selectScript failed:', e)
    alert('Failed to load script: ' + (e.response?.data?.error || e.message))
  }
}

async function delScript(s) {
  if (!confirm('Delete script?')) return
  try {
    await api.delete(`/api/scripts/${s.id}`)
    if (active.value?.id === s.id) { active.value = null; cmView.value?.destroy(); cmView.value = null }
    loadScripts()
  } catch (e) {
    console.error('deleteScript failed:', e)
  }
}

async function handleSave() {
  if (!active.value) return
  const s = active.value
  s.name = name.value
  s.label = label.value
  s.type = type.value
  if (cmView.value) s.tsCode = cmView.value.state.doc.toString()
  if (!s.name) return alert('Name required')
  if (!s.label) return alert('Label required')

  const payload = { name: s.name, label: s.label, type: s.type, tsCode: s.tsCode || '' }
  try {
    const res = s.id
      ? await api.put(`/api/scripts/${s.id}`, payload)
      : await api.post('/api/scripts', payload)
    active.value.id = res.data.id
    loadScripts()
  } catch (e) {
    console.error('save failed:', e)
    alert('Save failed: ' + (e.response?.data?.error || e.message))
  }
}

async function handleRun() {
  await handleSave()
  if (!active.value?.id) return
  outTab.value = 'output'
  output.value = 'Running...'; hasError.value = false
  try {
    const res = await api.post(`/api/scripts/${active.value.id}/run`)
    output.value = res.data.error || res.data.output || '(no output)'
    hasError.value = !!res.data.error
  } catch (e) {
    output.value = 'Error: ' + (e.response?.data?.error || e.message)
    hasError.value = true
  }
}

async function debug() {
  await handleSave()
  if (!active.value?.id) return
  output.value = 'Debugging...'; rowCount.value = ''
  try {
    const res = await api.post(`/api/scripts/${active.value.id}/debug`, { skip: 0 })
    handleDebugResult(res.data)
  } catch (e) {
    output.value = 'Error: ' + (e.response?.data?.error || e.message)
    hasError.value = true
  }
}

function handleDebugResult(data) {
  if (data.error) { output.value = data.error; hasError.value = true; return }
  output.value = data.output || '(no output)'; hasError.value = false
  if (data.breakpoints?.length) {
    rowCount.value = 'Hit #' + (data.hitCount || 1)
    let html = ''
    data.breakpoints.forEach(bp => {
      html += '<div style="margin-bottom:14px"><div style="color:#fbbf24;font-weight:600">● Line ' + bp.line + '</div>'
      if (bp.name) {
        html += '<div style="color:#94a3b8;font-size:11px;margin:4px 0">Stack:</div>'
        bp.name.replace(/\\n/g, '\n').split('\n').filter(s => s.trim()).forEach(s => {
          html += '<div style="color:#64748b;font-size:11px">' + esc(s.trim()) + '</div>'
        })
      }
      if (bp.vars && Object.keys(bp.vars).length) {
        html += '<div style="color:#94a3b8;font-size:11px;margin:4px 0">Variables:</div><table style="width:100%;font-size:11px">'
        Object.keys(bp.vars).sort().forEach(k => {
          let v = String(bp.vars[k]); if (v.length > 100) v = v.substring(0, 100) + '...'
          html += '<tr><td style="color:#60a5fa;width:120px">' + esc(k) + '</td><td>' + esc(v) + '</td></tr>'
        })
        html += '</table>'
      }
      html += '</div>'
    })
    debugHtml.value = html; outTab.value = 'debug'
  }
}

function esc(s) { return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;') }

async function loadBreakpoints() {
  if (!active.value?.id) return
  try { const res = await api.get('/api/scripts/' + active.value.id + '/breakpoints'); breakpoints = new Set(res.data.map(b => b.line)) } catch {}
}

function toggleBreakpoint(line) {
  if (!active.value?.id) return
  if (breakpoints.has(line)) { api.delete('/api/scripts/' + active.value.id + '/breakpoints/' + line); breakpoints.delete(line) }
  else { api.post('/api/scripts/' + active.value.id + '/breakpoints', { line, enabled: true }); breakpoints.add(line) }
}

function initEditor(code) {
  if (cmView.value) { cmView.value.destroy(); cmView.value = null }

  if (!editorHost.value) {
    console.error('editorHost ref is null — DOM not ready')
    return
  }

  const bpMarker = new class extends GutterMarker {
    toDOM() { const e = document.createElement('span'); e.textContent = '●'; e.style.color = '#f87171'; e.style.fontSize = '10px'; return e }
  }
  const bpGutter = gutter({
    class: 'cm-breakpoint-gutter',
    markers: (view) => {
      const m = []
      for (let i = 1; i <= view.state.doc.lines; i++) {
        if (breakpoints.has(i)) m.push(new bpMarker.range(i))
      }
      return m
    },
    initialSpacer: false,
  })

  const state = EditorState.create({
    doc: code || '',
    extensions: [
      lineNumbers(),
      bpGutter,
      javascript(),
      oneDark,
      history(),
      keymap.of([...defaultKeymap, indentWithTab,
        { key: 'Mod-s', run: () => { handleSave(); return true } },
        { key: 'Mod-Enter', run: () => { handleRun(); return true } },
      ]),
    ],
  })

  cmView.value = new EditorView({ state, parent: editorHost.value })

  cmView.value.dom.addEventListener('click', (e) => {
    if (!e.target.closest('.cm-breakpoint-gutter')) return
    if (!active.value?.id) return
    const v = cmView.value; if (!v) return
    const pos = v.posAtCoords({ x: e.clientX, y: e.clientY })
    if (pos == null) return
    toggleBreakpoint(v.state.doc.lineAt(pos).number)
  })
}
</script>
