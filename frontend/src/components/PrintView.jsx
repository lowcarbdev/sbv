import { useState, useEffect, useRef } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import axios from 'axios'
import { format } from 'date-fns'
import './PrintView.css'

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8081/api'

function PrintView() {
  const { address } = useParams()
  const [searchParams] = useSearchParams()
  const [messages, setMessages] = useState([])
  const [conversation, setConversation] = useState(null)
  const [loading, setLoading] = useState(true)
  const [mediaLoaded, setMediaLoaded] = useState(false)
  const [loadedCount, setLoadedCount] = useState(0)
  const [totalMedia, setTotalMedia] = useState(0)
  const printTriggeredRef = useRef(false)

  useEffect(() => {
    const startDate = searchParams.get('start')
    const endDate = searchParams.get('end')

    console.log('URL params:', { address, startDate, endDate })

    if (!address) {
      console.error('No address provided')
      return
    }

    fetchConversation(address, startDate, endDate)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const fetchConversation = async (address, startDate, endDate) => {
    try {
      setLoading(true)
      const params = { address, type: 'conversation' }
      if (startDate) params.start = startDate
      if (endDate) params.end = endDate

      console.log('Fetching conversation with params:', params)
      // Use /messages endpoint with type=conversation to get all types (SMS, MMS, calls)
      const response = await axios.get(`${API_BASE}/messages`, { params })
      const items = response.data || []

      console.log('Received items:', items.length)
      if (items.length > 0) {
        console.log('First item:', items[0])
        console.log('First item body:', items[0].body)
        console.log('First item date:', items[0].date)
        console.log('First item type:', items[0].type)
      }

      console.log('Setting messages state with items:', items)
      setMessages(items)
      console.log('Messages state set')

      // Get contact name from any message in the list (they should all have the same contact_name)
      const contactName = items.find(item => item.contact_name)?.contact_name || address
      console.log('Contact name:', contactName)

      setConversation({
        address,
        contactName
      })

      // Count total media items - need to check nested message for media_type
      const mediaCount = items.filter(item => {
        const msg = item.message || item
        return msg.media_type
      }).length
      setTotalMedia(mediaCount)
      console.log('Total media items:', mediaCount)

      setLoading(false)

      // Wait for all media to load before triggering print
      if (mediaCount > 0) {
        waitForAllMedia()
      } else {
        // No media, trigger print after short delay
        setTimeout(() => {
          if (!printTriggeredRef.current) {
            printTriggeredRef.current = true
            setMediaLoaded(true)
            window.print()
          }
        }, 500)
      }
    } catch (error) {
      console.error('Error fetching conversation:', error)
      setLoading(false)
    }
  }

  const waitForAllMedia = () => {
    const checkInterval = setInterval(() => {
      const images = document.querySelectorAll('.print-message-media img')
      const videos = document.querySelectorAll('.print-message-media video')
      const allMedia = [...images, ...videos]

      if (allMedia.length === 0) return

      const loaded = allMedia.filter(el => {
        if (el.tagName === 'IMG') {
          return el.complete && el.naturalHeight !== 0
        } else if (el.tagName === 'VIDEO') {
          return el.readyState >= 2
        }
        return false
      })

      setLoadedCount(loaded.length)

      // All media loaded
      if (loaded.length === allMedia.length) {
        clearInterval(checkInterval)
        clearTimeout(timeoutId)
        if (!printTriggeredRef.current) {
          printTriggeredRef.current = true
          setMediaLoaded(true)
          // Give browser a moment to render everything
          setTimeout(() => {
            window.print()
          }, 500)
        }
      }
    }, 100)

    // Timeout after 60 seconds
    const timeoutId = setTimeout(() => {
      clearInterval(checkInterval)
      if (!printTriggeredRef.current) {
        printTriggeredRef.current = true
        setMediaLoaded(true)
        window.print()
      }
    }, 60000)
  }

  const formatDate = (dateString) => {
    if (!dateString) return ''
    try {
      const date = new Date(dateString)
      return format(date, 'MMM d, yyyy h:mm a')
    } catch (e) {
      return ''
    }
  }

  const formatDuration = (seconds) => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  const getCallTypeInfo = (type) => {
    switch (type) {
      case 1: return { label: 'Incoming', icon: '↓' }
      case 2: return { label: 'Outgoing', icon: '↑' }
      case 3: return { label: 'Missed', icon: '✕' }
      case 4: return { label: 'Voicemail', icon: '⊙' }
      case 5: return { label: 'Rejected', icon: '✕' }
      case 6: return { label: 'Refused', icon: '✕' }
      default: return { label: 'Call', icon: '☎' }
    }
  }

  const renderMessage = (item) => {
    console.log('Rendering item:', {
      itemType: item.type,
      hasMessage: !!item.message,
      fullItem: item
    })

    // Check if this is a call at the Activity level
    if (item.type === 'call' && item.call) {
      // For calls in Activity items, the call data is nested in item.call
      const call = item.call
      const typeInfo = getCallTypeInfo(call.type)
      console.log('Rendering call:', {
        callId: call.id,
        callType: call.type,
        duration: call.duration,
        typeInfo
      })

      return (
        <div key={call.id} className="print-message print-call">
          <div className="print-message-bubble">
            <div className="print-call-info">
              <span className="print-call-icon">{typeInfo.icon}</span>
              <span className="print-call-label">{typeInfo.label} Call</span>
              {call.duration > 0 && (
                <span className="print-call-duration"> • {formatDuration(call.duration)}</span>
              )}
            </div>
            <div className="print-message-time">{formatDate(call.date)}</div>
          </div>
        </div>
      )
    }

    // For messages, extract from nested message object
    const message = item.message || item
    console.log('Rendering message:', {
      messageId: message.id,
      body: message.body,
      hasBody: !!(message.body && message.body !== ''),
      hasMedia: !!message.media_type
    })

    // Regular message rendering
    const isSent = message.type === 2
    const messageClass = isSent ? 'print-message sent' : 'print-message received'

    // Check if body has actual content (not null, not undefined, not empty string)
    const hasBody = message.body != null && message.body !== ''

    return (
      <div key={message.id} className={messageClass}>
        <div className="print-message-bubble">
          {hasBody && (
            <div className="print-message-body">{message.body}</div>
          )}
          {message.media_type && (
            <div className="print-message-media">
              {message.media_type.startsWith('image/') && (
                <img
                  src={`${API_BASE}/media?id=${message.id}`}
                  alt="Message attachment"
                  onLoad={() => console.log(`Image ${message.id} loaded`)}
                  onError={(e) => console.log(`Image ${message.id} failed to load:`, e)}
                />
              )}
              {message.media_type.startsWith('video/') && (
                <video
                  src={`${API_BASE}/media?id=${message.id}`}
                  controls
                  onLoadedData={() => console.log(`Video ${message.id} loaded`)}
                  onError={(e) => console.log(`Video ${message.id} failed to load:`, e)}
                />
              )}
              {message.media_type.startsWith('audio/') && (
                <div className="print-media-placeholder">
                  🎵 Audio attachment
                </div>
              )}
            </div>
          )}
          {!hasBody && !message.media_type && (
            <div className="print-message-body" style={{color: '#999', fontStyle: 'italic'}}>
              (Empty message)
            </div>
          )}
          <div className="print-message-time">{formatDate(message.date)}</div>
        </div>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="print-loading">
        <div className="spinner-border" role="status">
          <span className="visually-hidden">Loading...</span>
        </div>
        <p>Loading conversation...</p>
      </div>
    )
  }

  return (
    <div className="print-view">
      {!mediaLoaded && totalMedia > 0 && (
        <div className="print-loading-overlay">
          <div className="print-loading-content">
            <div className="spinner-border mb-3" role="status">
              <span className="visually-hidden">Loading...</span>
            </div>
            <h4>Preparing PDF...</h4>
            <p>Loading media: {loadedCount} of {totalMedia}</p>
            <div className="progress" style={{ width: '300px' }}>
              <div
                className="progress-bar"
                role="progressbar"
                style={{ width: `${(loadedCount / totalMedia) * 100}%` }}
                aria-valuenow={loadedCount}
                aria-valuemin="0"
                aria-valuemax={totalMedia}
              />
            </div>
          </div>
        </div>
      )}

      <div className="print-header">
        <h1>Conversation with {conversation?.contactName}</h1>
        <p className="print-address">{conversation?.address}</p>
        <p className="print-meta">
          {messages.length} items
          {' • '}
          Exported on {format(new Date(), 'MMMM d, yyyy')}
        </p>
      </div>

      <div className="print-messages">
        {(() => {
          console.log('About to render messages. Count:', messages.length)
          console.log('Messages array:', messages)
          console.log('Is array?', Array.isArray(messages))
          return messages.length > 0 ? (
            messages.map((msg, index) => {
              console.log(`Message ${index}:`, msg)
              return renderMessage(msg)
            })
          ) : (
            <p className="text-center text-muted">No messages to display</p>
          )
        })()}
      </div>
    </div>
  )
}

export default PrintView
