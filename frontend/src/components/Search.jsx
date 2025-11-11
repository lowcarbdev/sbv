import { useNavigate } from 'react-router-dom'
import { useEffect, useRef } from 'react'
import axios from 'axios'
import { format } from 'date-fns'

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8081/api'

function Search({ searchQuery, setSearchQuery, results, setResults, loading, setLoading, searched, setSearched, scrollPosition, setScrollPosition }) {
  const navigate = useNavigate()
  const scrollContainerRef = useRef(null)

  const handleSearch = async (e) => {
    e.preventDefault()
    if (!searchQuery.trim()) return

    setLoading(true)
    setSearched(true)

    try {
      const response = await axios.get(`${API_BASE}/search`, {
        params: { q: searchQuery, limit: 1000 }
      })
      setResults(response.data || [])
    } catch (error) {
      console.error('Error searching:', error)
      setResults([])
    } finally {
      setLoading(false)
    }
  }

  const handleResultClick = (result) => {
    // Navigate to the conversation with the message ID as a query parameter
    navigate(`/conversation/${encodeURIComponent(result.address)}?messageId=${result.message_id}`)
  }

  const formatPhoneNumber = (phoneNumber) => {
    if (!phoneNumber) return ''

    // Handle comma-separated numbers (group conversations)
    if (phoneNumber.includes(',')) {
      const numbers = phoneNumber.split(',').map(n => n.trim())
      return numbers.map(n => formatSinglePhoneNumber(n)).join(', ')
    }

    return formatSinglePhoneNumber(phoneNumber)
  }

  const formatSinglePhoneNumber = (phoneNumber) => {
    if (!phoneNumber) return ''

    // Remove +1 prefix if present
    const cleaned = phoneNumber.replace(/^\+1/, '')

    // Format as (XXX) XXX-XXXX
    if (cleaned.length === 10) {
      return `(${cleaned.slice(0, 3)}) ${cleaned.slice(3, 6)}-${cleaned.slice(6)}`
    }

    return phoneNumber
  }

  // Restore scroll position when component mounts or results change
  useEffect(() => {
    if (scrollContainerRef.current && scrollPosition > 0) {
      scrollContainerRef.current.scrollTop = scrollPosition
    }
  }, [scrollPosition])

  // Save scroll position when user scrolls
  const handleScroll = (e) => {
    if (e.target.scrollTop !== scrollPosition) {
      setScrollPosition(e.target.scrollTop)
    }
  }

  return (
    <div className="h-100 d-flex flex-column">
      {/* Header */}
      <div className="bg-light border-bottom p-3">
        <h2 className="h5 mb-3 d-flex align-items-center gap-2">
          <svg style={{width: '1.25rem', height: '1.25rem'}} className="text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          Search Messages
        </h2>

        <form onSubmit={handleSearch}>
          <div className="input-group">
            <input
              type="text"
              className="form-control"
              placeholder="Search message contents..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              autoFocus
            />
            <button
              type="submit"
              className="btn btn-primary"
              disabled={loading || !searchQuery.trim()}
            >
              {loading ? (
                <>
                  <span className="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                  Searching...
                </>
              ) : (
                'Search'
              )}
            </button>
          </div>
        </form>

        {searched && !loading && (
          <div className="mt-2 small text-muted">
            {results.length > 0 ? (
              <>
                Found <strong>{results.length.toLocaleString()}</strong> result{results.length !== 1 ? 's' : ''}
                {results.length >= 1000 && ' (limited to first 1000)'}
              </>
            ) : (
              'No results found'
            )}
          </div>
        )}
      </div>

      {/* Results */}
      <div ref={scrollContainerRef} onScroll={handleScroll} className="flex-fill overflow-auto p-3">
        {!searched ? (
          <div className="text-center text-muted py-5">
            <svg style={{width: '4rem', height: '4rem'}} className="mb-3 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <p className="lead">Search for messages</p>
            <p className="small">Enter a search term to find messages across all conversations</p>
          </div>
        ) : loading ? (
          <div className="text-center py-5">
            <div className="spinner-border text-primary mb-3" role="status">
              <span className="visually-hidden">Loading...</span>
            </div>
            <p className="text-muted">Searching...</p>
          </div>
        ) : results.length === 0 ? (
          <div className="text-center text-muted py-5">
            <svg style={{width: '4rem', height: '4rem'}} className="mb-3 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <p className="lead">No results found</p>
            <p className="small">Try a different search term</p>
          </div>
        ) : (
          <div className="row g-2">
            {results.map((result) => (
              <div key={result.message_id} className="col-12">
                <div
                  className="card h-100 shadow-sm"
                  style={{ cursor: 'pointer', transition: 'all 0.2s' }}
                  onClick={() => handleResultClick(result)}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.boxShadow = '0 0.5rem 1rem rgba(0,0,0,0.15)'
                    e.currentTarget.style.transform = 'translateY(-2px)'
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.boxShadow = ''
                    e.currentTarget.style.transform = ''
                  }}
                >
                  <div className="card-body">
                    <div className="d-flex justify-content-between align-items-start mb-2">
                      <div className="flex-fill">
                        <h6 className="card-title mb-1 fw-bold">
                          {result.contact_name || formatPhoneNumber(result.address) || 'Unknown'}
                        </h6>
                        {result.contact_name && (
                          <div className="small text-muted">
                            {formatPhoneNumber(result.address)}
                          </div>
                        )}
                      </div>
                      <small className="text-muted text-nowrap ms-2">
                        {format(new Date(result.date), 'MMM d, yyyy')}
                      </small>
                    </div>
                    <div
                      className="card-text small text-muted"
                      dangerouslySetInnerHTML={{ __html: result.snippet }}
                      style={{
                        wordBreak: 'break-word',
                        lineHeight: '1.5'
                      }}
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export default Search
