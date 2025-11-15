import { useState, useEffect } from 'react'
import { useNavigate, useLocation, Routes, Route } from 'react-router-dom'
import { Dropdown } from 'react-bootstrap'
import axios from 'axios'
import { useAuth } from './contexts/AuthContext'
import ConversationList from './components/ConversationList'
import MessageThread from './components/MessageThread'
import Activity from './components/Activity'
import Calls from './components/Calls'
import DateFilter from './components/DateFilter'
import Upload from './components/Upload'
import Search from './components/Search'
import ChangePasswordModal from './components/ChangePasswordModal'
import './App.css'

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8081/api'

function App() {
  const navigate = useNavigate()
  const location = useLocation()
  const { user, logout } = useAuth()
  const [conversations, setConversations] = useState([])
  const [conversationsLoading, setConversationsLoading] = useState(false)
  const [selectedConversation, setSelectedConversation] = useState(null)
  const [startDate, setStartDate] = useState(null)
  const [endDate, setEndDate] = useState(null)
  const [dateRange, setDateRange] = useState({ min: null, max: null })
  const [showUpload, setShowUpload] = useState(false)
  const [showPasswordModal, setShowPasswordModal] = useState(false)
  const [searchFilter, setSearchFilter] = useState('')

  // Search state (persisted across tab switches)
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState([])
  const [searchLoading, setSearchLoading] = useState(false)
  const [searchExecuted, setSearchExecuted] = useState(false)
  const [searchScrollPosition, setSearchScrollPosition] = useState(0)

  // Derive activeView from URL
  const activeView = location.pathname.startsWith('/activity')
    ? 'activity'
    : location.pathname.startsWith('/calls')
    ? 'calls'
    : location.pathname.startsWith('/search')
    ? 'search'
    : 'conversations'

  useEffect(() => {
    fetchDateRange()
    fetchConversations()
  }, [])

  useEffect(() => {
    fetchConversations()
  }, [startDate, endDate])

  // Sync selected conversation from URL
  useEffect(() => {
    const match = location.pathname.match(/^\/conversation\/(.+)$/)
    if (match) {
      const address = decodeURIComponent(match[1])
      // Find conversation by address
      const conversation = conversations.find(c => c.address === address)
      if (conversation) {
        setSelectedConversation(conversation)
      } else if (conversations.length > 0) {
        // If conversation not found in list, create a minimal conversation object
        setSelectedConversation({ address, contact_name: address, type: 'message' })
      }
    } else if (location.pathname === '/' || location.pathname === '/conversations') {
      setSelectedConversation(null)
    }
  }, [location.pathname, conversations])

  const fetchDateRange = async () => {
    try {
      const response = await axios.get(`${API_BASE}/daterange`)
      setDateRange({
        min: new Date(response.data.min_date),
        max: new Date(response.data.max_date)
      })
    } catch (error) {
      console.error('Error fetching date range:', error)
    }
  }

  const fetchConversations = async () => {
    setConversationsLoading(true)
    try {
      const params = {}
      if (startDate) params.start = startDate.toISOString()
      if (endDate) params.end = endDate.toISOString()

      const response = await axios.get(`${API_BASE}/conversations`, { params })
      setConversations(response.data || [])
    } catch (error) {
      console.error('Error fetching conversations:', error)
    } finally {
      setConversationsLoading(false)
    }
  }

  const handleUploadSuccess = () => {
    setShowUpload(false)
    fetchDateRange()
    fetchConversations()
  }

  const handleSelectConversation = (conversation) => {
    if (conversation) {
      navigate(`/conversation/${encodeURIComponent(conversation.address)}`)
    }
  }

  const handleViewChange = (view) => {
    if (view === 'activity') {
      navigate('/activity')
    } else if (view === 'calls') {
      navigate('/calls')
    } else if (view === 'search') {
      navigate('/search')
    } else {
      navigate('/')
    }
  }

  // Filter conversations based on search text
  const filteredConversations = conversations.filter(conv => {
    if (!searchFilter) return true

    const searchLower = searchFilter.toLowerCase()
    const nameMatch = conv.contact_name && conv.contact_name.toLowerCase().includes(searchLower)
    const addressMatch = conv.address && conv.address.toLowerCase().includes(searchLower)

    return nameMatch || addressMatch
  })

  return (
    <div className="vh-100 d-flex flex-column bg-light">
      {/* Header */}
      <header className="bg-primary bg-gradient text-white p-2 shadow-lg">
        <div className="d-flex justify-content-between align-items-center">
          <div className="d-flex align-items-center gap-3">
            <svg style={{width: '2.5rem', height: '2.5rem'}} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
            </svg>
            <div>
              <h1 className="h2 mb-1 fw-bold">SMS Backup Viewer</h1>
              <p className="mb-0 small opacity-75">View and browse your message history</p>
            </div>
          </div>
          <div className="d-flex align-items-center gap-3">
            <div className="text-end">
              <div className="small opacity-75">Logged in as</div>
              <div className="fw-bold">{user?.username}</div>
            </div>
            <button
              onClick={() => setShowUpload(true)}
              className="btn btn-light btn-lg shadow d-flex align-items-center gap-2"
            >
              <svg style={{width: '1.25rem', height: '1.25rem'}} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
              </svg>
              Upload Backup
            </button>
            <Dropdown align="end">
              <Dropdown.Toggle
                variant="outline-light"
                className="d-flex align-items-center gap-2"
                style={{ backgroundColor: 'transparent', borderColor: 'rgba(255, 255, 255, 0.5)' }}
              >
                <svg style={{width: '1.5rem', height: '1.5rem'}} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                </svg>
              </Dropdown.Toggle>
              <Dropdown.Menu>
                <Dropdown.ItemText className="small text-muted">
                  Version {__APP_VERSION__}
                </Dropdown.ItemText>
                <Dropdown.Divider />
                <Dropdown.Item onClick={() => setShowPasswordModal(true)}>
                  <svg style={{width: '1rem', height: '1rem'}} className="me-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                  </svg>
                  Change Password
                </Dropdown.Item>
                <Dropdown.Divider />
                <Dropdown.Item onClick={logout}>
                  <svg style={{width: '1rem', height: '1rem'}} className="me-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                  </svg>
                  Logout
                </Dropdown.Item>
              </Dropdown.Menu>
            </Dropdown>
          </div>
        </div>
      </header>

      {/* View Switcher */}
      <div className="bg-white border-bottom shadow-sm">
        <div className="container-fluid">
          <ul className="nav nav-tabs border-0">
            <li className="nav-item">
              <button
                className={`nav-link ${activeView === 'conversations' ? 'active' : ''}`}
                onClick={() => handleViewChange('conversations')}
              >
                <svg style={{width: '1rem', height: '1rem'}} className="me-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
                </svg>
                Conversations
              </button>
            </li>
            <li className="nav-item">
              <button
                className={`nav-link ${activeView === 'calls' ? 'active' : ''}`}
                onClick={() => handleViewChange('calls')}
              >
                <svg style={{width: '1rem', height: '1rem'}} className="me-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
                </svg>
                Calls
              </button>
            </li>
            <li className="nav-item">
              <button
                className={`nav-link ${activeView === 'search' ? 'active' : ''}`}
                onClick={() => handleViewChange('search')}
              >
                <svg style={{width: '1rem', height: '1rem'}} className="me-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                Search
              </button>
            </li>
            <li className="nav-item">
              <button
                className={`nav-link ${activeView === 'activity' ? 'active' : ''}`}
                onClick={() => handleViewChange('activity')}
              >
                <svg style={{width: '1rem', height: '1rem'}} className="me-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                Activity
              </button>
            </li>
          </ul>
        </div>
      </div>

      {/* Date Filter */}
      <div className="bg-white border-bottom shadow-sm">
        <DateFilter
          startDate={startDate}
          endDate={endDate}
          minDate={dateRange.min}
          maxDate={dateRange.max}
          onStartDateChange={setStartDate}
          onEndDateChange={setEndDate}
        />
      </div>

      {/* Main Content */}
      <div className="flex-fill d-flex overflow-hidden gap-2 p-2">
        {activeView === 'conversations' ? (
          <>
            {/* Conversation List */}
            <div style={{width: '380px', minWidth: '380px', maxWidth: '380px', flexShrink: 0}} className="bg-white rounded-3 shadow overflow-hidden border">
              <div className="bg-light border-bottom p-2">
                <h2 className="h5 mb-2 d-flex align-items-center gap-2">
                  <svg style={{width: '1.25rem', height: '1.25rem'}} className="text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
                  </svg>
                  Conversations
                </h2>
                <div className="position-relative">
                  <svg style={{width: '1rem', height: '1rem', position: 'absolute', left: '0.75rem', top: '50%', transform: 'translateY(-50%)'}} className="text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                  <input
                    type="text"
                    className="form-control form-control-sm ps-5"
                    placeholder="Search by name or number..."
                    value={searchFilter}
                    onChange={(e) => setSearchFilter(e.target.value)}
                  />
                </div>
              </div>
              <div className="overflow-auto" style={{height: 'calc(100% - 7rem)'}}>
                <ConversationList
                  conversations={filteredConversations}
                  selectedConversation={selectedConversation}
                  onSelectConversation={handleSelectConversation}
                  loading={conversationsLoading}
                />
              </div>
            </div>

            {/* Message Thread */}
            <div className="flex-fill bg-white rounded-3 shadow overflow-hidden border" style={{minWidth: 0}}>
              <MessageThread
                conversation={selectedConversation}
                startDate={startDate}
                endDate={endDate}
              />
            </div>
          </>
        ) : activeView === 'search' ? (
          /* Search View */
          <div className="flex-fill bg-white rounded-3 shadow overflow-hidden border" style={{minWidth: 0}}>
            <Search
              searchQuery={searchQuery}
              setSearchQuery={setSearchQuery}
              results={searchResults}
              setResults={setSearchResults}
              loading={searchLoading}
              setLoading={setSearchLoading}
              searched={searchExecuted}
              setSearched={setSearchExecuted}
              scrollPosition={searchScrollPosition}
              setScrollPosition={setSearchScrollPosition}
            />
          </div>
        ) : activeView === 'calls' ? (
          /* Calls View */
          <div className="flex-fill bg-white rounded-3 shadow overflow-hidden border" style={{minWidth: 0}}>
            <Calls
              startDate={startDate}
              endDate={endDate}
            />
          </div>
        ) : (
          /* Activity View */
          <div className="flex-fill bg-white rounded-3 shadow overflow-hidden border" style={{minWidth: 0}}>
            <Activity
              startDate={startDate}
              endDate={endDate}
            />
          </div>
        )}
      </div>

      {/* Upload Modal */}
      {showUpload && (
        <Upload
          onClose={() => setShowUpload(false)}
          onSuccess={handleUploadSuccess}
        />
      )}

      {/* Change Password Modal */}
      {showPasswordModal && (
        <ChangePasswordModal
          onClose={() => setShowPasswordModal(false)}
          onSuccess={() => {
            // Password changed successfully
            console.log('Password changed successfully')
          }}
        />
      )}
    </div>
  )
}

export default App
