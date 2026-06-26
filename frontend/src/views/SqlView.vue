<template>
  <div class="sql-layout">
    <aside class="sql-sidebar">
      <div class="sidebar-hd">Tables</div>
      <ul class="sql-sidebar-list">
        <li v-for="t in tables" :key="t" @click="selectTable(t)">{{ t }}</li>
      </ul>
    </aside>
    <main class="sql-main">
      <div class="sql-toolbar">
        <span>SQL Query · ⌘/Ctrl + Enter to run</span>
        <button class="btn btn-info btn-sm" @click="runAll">▸ Run</button>
      </div>
      <div class="sql-editor-wrap" ref="sqlHost"></div>
      <div class="sql-result">
        <div v-if="error" class="output-content error">{{ error }}</div>
        <table v-else-if="columns.length">
          <thead><tr><th v-for="c in columns" :key="c">{{ c }}</th></tr></thead>
          <tbody>
            <tr v-for="(row,i) in rows" :key="i"><td v-for="(cell,j) in row" :key="j">{{ cell === null ? 'NULL' : cell }}</td></tr>
          </tbody>
        </table>
        <div v-else class="empty-state">{{ status || 'Select a table or write SQL' }}</div>
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
const columns = ref([])
const rows = ref([])
const error = ref('')
const status = ref('')
const sqlHost = ref(null)
const sqlView = shallowRef(null)

onMounted(async () => {
  try {
    const res = await api.get('/api/sql/tables')
    tables.value = res.data
  } catch {}
  nextTick(initEditor)
})

function initEditor() {
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
  error.value = ''; columns.value = []; rows.value = []; status.value = 'Running...'
  try {
    const res = await api.post('/api/sql/run', { sql: sqlText })
    if (res.data.error) { error.value = res.data.error; return }
    columns.value = res.data.columns || []; rows.value = res.data.rows || []
    status.value = `(${res.data.rowCount || 0} rows)`
  } catch (e) { error.value = e.response?.data?.error || 'Query error' }
}
</script>
