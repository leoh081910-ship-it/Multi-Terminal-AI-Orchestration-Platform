import { useState } from 'react'
import { Layout } from './components/Layout'
import { TaskList } from './components/TaskList'
import './App.css'

type Tab = 'tasks'

function App() {
  const [activeTab, setActiveTab] = useState<Tab>('tasks')

  return (
    <Layout activeTab={activeTab} onTabChange={setActiveTab}>
      <TaskList />
    </Layout>
  )
}

export default App
