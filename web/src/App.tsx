import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { ProjectListPage } from './pages/ProjectListPage'
import { DashboardPage } from './pages/DashboardPage'
import { BoardPage } from './pages/BoardPage'
import { ListPage } from './pages/ListPage'
import { TicketDetailPage } from './pages/TicketDetailPage'
import { DocumentPage } from './pages/DocumentPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<ProjectListPage />} />
        <Route path="/projects/:project" element={<Layout />}>
          <Route index element={<DashboardPage />} />
          <Route path="board" element={<BoardPage />} />
          <Route path="list" element={<ListPage />} />
          <Route path="tickets/:ticketId" element={<TicketDetailPage />} />
          <Route path="docs/*" element={<DocumentPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
