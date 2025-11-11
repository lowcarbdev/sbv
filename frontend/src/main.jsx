import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import axios from 'axios'
import 'bootstrap/dist/css/bootstrap.min.css'
import './index.css'
import { AuthProvider } from './contexts/AuthContext'
import App from './App.jsx'
import Login from './components/Login.jsx'
import ProtectedRoute from './components/ProtectedRoute.jsx'

// Configure axios to include credentials with all requests
axios.defaults.withCredentials = true

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/*" element={
            <ProtectedRoute>
              <App />
            </ProtectedRoute>
          } />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  </StrictMode>,
)
