<template>
  <div class="editor-layout">
    <aside class="editor-sidebar">
      <div class="sidebar-hd">
        <h3>Scripts</h3>
        <button class="btn-blue" style="padding:3px 10px;font-size:11px" @click="newScript">+ New</button>
      </div>
      <ul class="list">
        <li v-for="s in scripts" :key="s.id" :class="{ active: active?.id === s.id }" @click="selectScript(s)">
          <div>
            <div class="name">{{ s.label || s.name }}</div>
            <div class="meta">{{ s.name }}{{ s.type ? ' · ' + s.type : '' }}</div>
          </div>
          <button class="del" @click.stop="deleteScript(s)">×</button>
        </li>
      </ul>
    </aside>
    <main class="editor-main">
      <div class="editor-toolbar">
        <input v-model="name" placeholder="name (unique)" />
        <input v-model="label" placeholder="display name" />
        <input v-model="type" placeholder="type" style="width:100px" />
        <div class="spacer"></div>
        <button class="btn-yellow" @click="debug" v-if="active">Debug</button>
        <button class="btn-green" @click="save" v-if="active">Save</button>
        <button class="btn-blue" @click="run" v-if="active">Run</button>
      </div>
      <div class="cm-wrap" ref="editorHost"></div>
      <div class="editor-output">
        <div class="output-hd">
          <span :class="{ active: outTab==='output' }" @click="outTab='output'">Output</span>
          <span :class="{ active: outTab==='debug' }" @click="outTab='debug'">Debug</span>
          <span style="margin-left:auto;color:var(--muted)">{{ rowCount }}</span>
        </div>
        <pre v-if="outTab==='output'" :class="{ 'output-error': hasError }" v-text="output || '(no output)'"></pre>
        <div v-else style="padding:10px 16px;font-size:12px" v-html="debugHtml"></div>
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
import { defaultKeymap, history } from '@codemirror/commands'

const scripts = ref([])
const active = ref(null)
const name = ref('')
const label = ref('')
const type = ref('')
const output = ref('')
const hasError = ref(false)
const outTab = ref('output')
const rowCount = ref('')
const debugHtml = ref('')
const editorHost = ref(null)
const cmView = shallowRef(null)
let breakpoints = new Set()

onMounted(() => loadScripts())

async function loadScripts() {
  const res = await api.get('/api/scripts')
  scripts.value = res.data
}

function newScript() {
  if (active.value) saveForm()
  active.value = { id: null, name: '', label: 'Untitled', type: '', tsCode: '' }
  name.value = ''; label.value = 'Untitled'; type.value = ''
  initEditor('')
}

async function selectScript(s) {
  if (active.value) saveForm()
  const res = await api.get(`/api/scripts/${s.id}`)
  active.value = res.data
  name.value = res.data.name; label.value = res.data.label; type.value = res.data.type
  initEditor(res.data.tsCode || '')
  loadBreakpoints()
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
  if (!s.name) return alert('Name is required')
  if (!s.label) return alert('Label is required')
  const payload = { name: s.name, label: s.label, type: s.type, tsCode: s.tsCode }
  const res = s.id
    ? await api.put(`/api/scripts/${s.id}`, payload)
    : await api.post('/api/scripts', payload)
  active.value.id = res.data.id
  loadScripts()
}

async function run() {
  await save()
  if (!active.value?.id) return
  outTab.value = 'output'
  output.value = 'Running...'
  hasError.value = false
  try {
    const res = await api.post(`/api/scripts/${active.value.id}/run`)
    output.value = res.data.error || res.data.output || '(no output)'
    hasError.value = !!res.data.error
  } catch { output.value = 'Error'; hasError.value = true }
}

async function debug() {
  await save()
  if (!active.value?.id) return
  outTab.value = 'output'
  output.value = 'Debugging...'
  rowCount.value = ''
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
      html += `<div style="margin-bottom:12px"><div style="color:#f9e2af;font-weight:600">● Line ${bp.line}</div>`
      if (bp.name) {
        html += `<div style="color:#a6adc8;font-size:11px;margin:4px 0">Stack:</div>`
        bp.name.replace(/\\n/g,'\n').split('\n').filter(s=>s.trim()).forEach(s => {
          html += `<div style="color:#585b70;font-size:11px">${esc(s.trim())}</div>`
        })
      }
      if (bp.vars && Object.keys(bp.vars).length) {
        html += `<div style="color:#a6adc8;font-size:11px;margin:4px 0">Vars:</div><table style="width:100%;font-size:11px">`
        Object.keys(bp.vars).sort().forEach(k => {
          let v = String(bp.vars[k]); if (v.length > 100) v = v.substring(0,100)+'...'
          html += `<tr><td style="color:#89b4fa;width:120px">${esc(k)}</td><td>${esc(v)}</td></tr>`
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
  if (!confirm('Delete this script?')) return
  await api.delete(`/api/scripts/${s.id}`)
  if (active.value?.id === s.id) { active.value = null; name.value = ''; label.value = ''; type.value = ''; cmView.value?.destroy(); cmView.value = null }
  loadScripts()
}

async function loadBreakpoints() {
  if (!active.value?.id) return
  const res = await api.get(`/api/scripts/${active.value.id}/breakpoints`)
  breakpoints = new Set(res.data.map(b => b.line))
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

  const bpMarker = new class extends GutterMarker { toDOM() { const e = document.createElement('span'); e.textContent = '●'; e.style.color = '#f38ba8'; return e } }
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
      lineNumbers(),
      bpGutter,
      javascript(),
      oneDark,
      history(),
      keymap.of([...defaultKeymap,
        { key: 'Ctrl-s', run: () => { save(); return true } },
        { key: 'Cmd-s', run: () => { save(); return true } },
        { key: 'Ctrl-Enter', run: () => { run(); return true } },
        { key: 'Cmd-Enter', run: () => { run(); return true } },
      ]),
      EditorView.updateListener.of(update => {
        if (active.value) active.value.tsCode = update.state.doc.toString()
      }),
    ],
  })

  cmView.value = new EditorView({
    state,
    parent: editorHost.value,
    dispatch: (tr) => {
      cmView.value.update([tr])
      // gutter click → toggle breakpoint
      if (tr.isUserEvent('select.gutter') && tr.startState.selection) {
        const pos = tr.startState.selection.main.head
        const line = tr.startState.doc.lineAt(pos).number
        setTimeout(() => toggleBreakpoint(line), 0)
      }
    }
  })
}
</script>
