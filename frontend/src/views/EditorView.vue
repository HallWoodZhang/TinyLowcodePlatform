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
          <button class="si-del" @click.stop="deleteScript(s)">×</button>
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
        <button class="btn btn-success btn-sm" @click="save">Save</button>
        <button class="btn btn-info btn-sm" @click="run">▸ Run</button>
      </div>
      <div class="editor-toolbar" v-else style="color:var(--text-muted);justify-content:center">
        Select a script from the sidebar or create a new one
      </div>
      <div class="cm-wrap" ref="editorHost"></div>
      <div class="editor-output">
        <div class="output-tabs">
          <span :class="{ active: outTab==='output' }" @click="outTab='output'">Output</span>
          <span :class="{ active: outTab==='debug' }" @click="outTab='debug'">Debug</span>
          <span style="margin-left:auto;color:var(--text-muted)">{{ rowCount }}</span>
        </div>
        <div v-if="outTab==='output'" class="output-content" :class="{ error: hasError }" v-text="output || '(no output yet)'"></div>
        <div v-else class="output-content" v-html="debugHtml"></div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted, shallowRef } from 'vue'
import api from '../utils/api'
import { EditorState } from '@codemirror/state'
import { EditorView, keymap, lineNumbers, gutter, GutterMarker } from '@codemirror/view'
import { javascript } from '@codemirror/lang-javascript'
import { oneDark } from '@codemirror/theme-one-dark'
import { defaultKeymap, history } from '@codemirror/commands'

const scripts = ref([])
const active = ref(null)
const name = ref('')
const label = ref('')
const type = ref('')
const output = ref('(no output yet)')
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
    scripts.value = res.data
  } catch {}
}

function newScript() {
  saveForm()
  active.value = { id: null, name: '', label: 'Untitled', type: '', tsCode: '' }
  name.value = ''; label.value = 'Untitled'; type.value = ''
  initEditor('')
}

async function selectScript(s) {
  saveForm()
  try {
    const res = await api.get(`/api/scripts/${s.id}`)
    active.value = res.data
    name.value = res.data.name; label.value = res.data.label; type.value = res.data.type
    initEditor(res.data.tsCode || '')
    loadBreakpoints()
  } catch {}
}

function saveForm() {
  if (!active.value) return
  active.value.name = name.value; active.value.label = label.value; active.value.type = type.value
  active.value.tsCode = cmView.value?.state.doc.toString() || ''
}

async function save() {
  if (!active.value) return
  saveForm()
  const s = active.value
  if (!s.name) return alert('Name required')
  if (!s.label) return alert('Label required')
  const p = { name: s.name, label: s.label, type: s.type, tsCode: s.tsCode }
  try {
    const res = s.id ? await api.put(`/api/scripts/${s.id}`, p) : await api.post('/api/scripts', p)
    active.value.id = res.data.id
    loadScripts()
  } catch {}
}

async function run() {
  await save()
  if (!active.value?.id) return
  outTab.value = 'output'
  output.value = 'Running...'; hasError.value = false
  try {
    const res = await api.post(`/api/scripts/${active.value.id}/run`)
    output.value = res.data.error || res.data.output || '(no output)'
    hasError.value = !!res.data.error
  } catch { output.value = 'Error'; hasError.value = true }
}

async function debug() {
  await save()
  if (!active.value?.id) return
  output.value = 'Debugging...'; rowCount.value = ''
  try {
    const res = await api.post(`/api/scripts/${active.value.id}/debug`, { skip: 0 })
    handleDebugResult(res.data)
  } catch { output.value = 'Error'; hasError.value = true }
}

function handleDebugResult(data) {
  if (data.error) { output.value = data.error; hasError.value = true; return }
  output.value = data.output || '(no output)'
  hasError.value = false
  if (data.breakpoints?.length) {
    rowCount.value = 'Hit #' + (data.hitCount || 1)
    let html = ''
    data.breakpoints.forEach(bp => {
      html += `<div style="margin-bottom:14px"><div style="color:#fbbf24;font-weight:600;font-size:13px">● Line ${bp.line}</div>`
      if (bp.name) {
        html += `<div style="color:#94a3b8;font-size:11px;margin:4px 0">Stack:</div>`
        bp.name.replace(/\\n/g,'\n').split('\n').filter(s=>s.trim()).forEach(s => {
          html += `<div style="color:#64748b;font-size:11px">${esc(s.trim())}</div>`
        })
      }
      if (bp.vars && Object.keys(bp.vars).length) {
        html += `<div style="color:#94a3b8;font-size:11px;margin:4px 0">Variables:</div><table style="width:100%;font-size:11px">`
        Object.keys(bp.vars).sort().forEach(k => {
          let v = String(bp.vars[k]); if (v.length > 100) v = v.substring(0,100)+'...'
          html += `<tr><td style="color:#60a5fa;width:120px;padding:2px 4px">${esc(k)}</td><td style="padding:2px 4px">${esc(v)}</td></tr>`
        })
        html += '</table>'
      }
      html += '</div>'
    })
    debugHtml.value = html
    outTab.value = 'debug'
  }
}

function esc(s) { return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;') }

async function deleteScript(s) {
  if (!confirm('Delete script?')) return
  await api.delete(`/api/scripts/${s.id}`)
  if (active.value?.id === s.id) { active.value = null; name.value = ''; label.value = ''; type.value = ''; cmView.value?.destroy(); cmView.value = null }
  loadScripts()
}

async function loadBreakpoints() {
  if (!active.value?.id) return
  try {
    const res = await api.get(`/api/scripts/${active.value.id}/breakpoints`)
    breakpoints = new Set(res.data.map(b => b.line))
  } catch {}
}

function toggleBreakpoint(line) {
  if (!active.value?.id) return
  if (breakpoints.has(line)) {
    api.delete(`/api/scripts/${active.value.id}/breakpoints/${line}`)
    breakpoints.delete(line)
  } else {
    api.post(`/api/scripts/${active.value.id}/breakpoints`, { line, enabled: true })
    breakpoints.add(line)
  }
}

function initEditor(code) {
  if (cmView.value) cmView.value.destroy()

  const bpMarker = new class extends GutterMarker {
    toDOM() { const e = document.createElement('span'); e.textContent = '●'; e.style.color = '#f87171'; e.style.fontSize = '10px'; return e }
  }
  const bpGutter = gutter({
    class: 'cm-breakpoint-gutter',
    markers: (view) => {
      const markers = []
      for (let i = 1; i <= view.state.doc.lines; i++) {
        if (breakpoints.has(i)) markers.push(new bpMarker.range(i))
      }
      return markers
    },
    initialSpacer: false,
  })

  const state = EditorState.create({
    doc: code,
    extensions: [
      lineNumbers(), bpGutter, javascript(), oneDark, history(),
      keymap.of([...defaultKeymap,
        { key: 'Ctrl-s', run: () => { save(); return true } },
        { key: 'Cmd-s', run: () => { save(); return true } },
        { key: 'Ctrl-Enter', run: () => { run(); return true } },
        { key: 'Cmd-Enter', run: () => { run(); return true } },
      ]),
    ],
  })

  cmView.value = new EditorView({
    state, parent: editorHost.value,
    dispatch: (tr) => {
      cmView.value.update([tr])
      if (tr.isUserEvent('select.gutter')) {
        const pos = tr.startState.selection?.main?.head
        if (pos != null) {
          const line = tr.startState.doc.lineAt(pos).number
          setTimeout(() => toggleBreakpoint(line), 0)
        }
      }
    },
  })
}
</script>
