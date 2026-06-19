<template>
  <div class="sql-layout">
    <aside class="sql-sidebar">
      <div class="sidebar-hd">Tables</div>
      <ul class="list">
        <li v-for="t in tables" :key="t" @click="selectTable(t)">{{ t }}</li>
      </ul>
    </aside>
    <main class="sql-main">
      <div class="sql-toolbar">
        <span>SQL Query — Ctrl+Enter to run</span>
        <div style="display:flex;gap:6px">
          <button class="btn-blue" style="padding:4px 12px;font-size:11px" @click="runAll">Run</button>
        </div>
      </div>
      <div class="sql-editor" ref="sqlHost"></div>
      <div class="sql-result">
        <div v-if="resultError" class="error-text">{{ resultError }}</div>
        <table v-else-if="resultColumns.length">
          <thead><tr><th v-for="c in resultColumns" :key="c">{{ c }}</th></tr></thead>
          <tbody>
            <tr v-for="(row,i) in resultRows" :key="i"><td v-for="(cell,j) in row" :key="j">{{ cell === null ? 'NULL' : cell }}</td></tr>
          </tbody>
        </table>
        <div v-else class="muted">{{ resultMsg }}</div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, shallowRef, onMounted, nextTick } from 'vue'
import api from '../utils/api'
import { EditorState } from '@codemirror/state'
import { EditorView, keymap, lineNumbers } from '@codemirror/view'
import { sql } from '@codemirror/lang-sql'
import { oneDark } from '@codemirror/theme-one-dark'
import { defaultKeymap } from '@codemirror/commands'

const tables = ref([])
const resultColumns = ref([])
const resultRows = ref([])
const resultError = ref('')
const resultMsg = ref('Select a table or write SQL')
const sqlHost = ref(null)
const sqlView = shallowRef(null)

onMounted(async () => {
  const res = await api.get('/api/sql/tables')
  tables.value = res.data
  nextTick(initSqlEditor)
})

function initSqlEditor() {
  const state = EditorState.create({
    doc: '',
    extensions: [
      lineNumbers(), sql(), oneDark,
      keymap.of([...defaultKeymap,
        { key: 'Ctrl-Enter', run: () => { runAll(); return true } },
        { key: 'Cmd-Enter', run: () => { runAll(); return true } },
      ]),
    ],
  })
  sqlView.value = new EditorView({ state, parent: sqlHost.value })
}

function selectTable(name) {
  sqlView.value.dispatch({ changes: { from: 0, to: sqlView.value.state.doc.length, insert: `SELECT * FROM ${name} LIMIT 100;` } })
  sqlView.value.focus()
}

async function runAll() {
  const sqlText = sqlView.value.state.doc.toString().trim()
  if (!sqlText) return
  resultError.value = ''; resultColumns.value = []; resultRows.value = []; resultMsg.value = 'Running...'
  try {
    const res = await api.post('/api/sql/run', { sql: sqlText })
    if (res.data.error) { resultError.value = res.data.error; return }
    resultColumns.value = res.data.columns || []
    resultRows.value = res.data.rows || []
    resultMsg.value = `(${res.data.rowCount || 0} rows)`
  } catch (e) { resultError.value = e.response?.data?.error || 'Error' }
}
</script>
