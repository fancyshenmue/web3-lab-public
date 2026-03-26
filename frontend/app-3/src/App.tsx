import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { HomePage } from './pages/HomePage'
import { LoginPage } from './pages/LoginPage'
import { CallbackPage } from './pages/CallbackPage'
import { DashboardPage } from './pages/DashboardPage'

export default function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-gray-50 font-sans text-gray-900">
        <Routes>
          <Route path="/" element={<div className="flex flex-col justify-center items-center min-h-screen p-4"><div className="max-w-md w-full space-y-8 bg-white p-10 rounded-xl shadow-lg border border-gray-100"><HomePage /></div></div>} />
          <Route path="/login" element={<div className="flex flex-col justify-center items-center min-h-screen p-4"><div className="max-w-md w-full space-y-8 bg-white p-10 rounded-xl shadow-lg border border-gray-100"><LoginPage /></div></div>} />
          <Route path="/callback" element={<div className="flex flex-col justify-center items-center min-h-screen p-4"><div className="max-w-md w-full space-y-8 bg-white p-10 rounded-xl shadow-lg border border-gray-100"><CallbackPage /></div></div>} />
          <Route path="/profile" element={<Navigate to="/dashboard" replace />} />
          
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/logout" element={<Navigate to="/?logout=true" replace />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </div>
    </BrowserRouter>
  )
}
