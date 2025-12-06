import { useState, useEffect } from 'react'
import axios from 'axios'
import MediaCarousel from './MediaCarousel'
import './MediaGrid.css'

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8081/api'

function MediaGrid({ conversation, startDate, endDate }) {
  const [mediaItems, setMediaItems] = useState([])
  const [loading, setLoading] = useState(false)
  const [selectedIndex, setSelectedIndex] = useState(null)
  const [failedVideos, setFailedVideos] = useState(new Set())
  const [transcodeVideos, setTranscodeVideos] = useState(new Set())

  useEffect(() => {
    fetchMediaItems()
    setFailedVideos(new Set()) // Reset failed videos when conversation changes
    setTranscodeVideos(new Set()) // Reset transcode videos when conversation changes
  }, [conversation, startDate, endDate])

  const fetchMediaItems = async () => {
    setLoading(true)
    try {
      const params = {
        address: conversation.address
      }
      if (startDate) params.start = startDate.toISOString()
      if (endDate) params.end = endDate.toISOString()

      const response = await axios.get(`${API_BASE}/media-items`, { params })
      setMediaItems(response.data || [])
    } catch (error) {
      console.error('Error fetching media items:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleThumbnailClick = (index) => {
    setSelectedIndex(index)
  }

  const handleCloseCarousel = () => {
    setSelectedIndex(null)
  }

  const handleVideoError = (itemId, alreadyTranscoded = false) => {
    if (!alreadyTranscoded) {
      console.log('Video failed to play, retrying with transcoding:', itemId)
      // Mark this video to be transcoded and it will reload
      setTranscodeVideos(prev => new Set([...prev, itemId]))
      return
    }
    // If already transcoded and still failing, hide it
    console.warn('Video not playable even after transcoding:', itemId)
    setFailedVideos(prev => new Set([...prev, itemId]))
  }

  // Filter out failed videos from display
  const displayableItems = mediaItems.filter(item => !failedVideos.has(item.id))

  if (loading) {
    return (
      <div className="d-flex justify-content-center align-items-center" style={{ minHeight: '400px' }}>
        <div className="spinner-border text-primary" role="status">
          <span className="visually-hidden">Loading media...</span>
        </div>
      </div>
    )
  }

  if (displayableItems.length === 0 && !loading) {
    return (
      <div className="text-center py-5">
        <svg
          style={{ width: '4rem', height: '4rem' }}
          className="mb-3 text-muted opacity-50"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
          />
        </svg>
        <p className="text-muted">
          {mediaItems.length === 0
            ? 'No photos or videos found in this conversation'
            : 'No browser-compatible photos or videos found in this conversation'}
        </p>
      </div>
    )
  }

  return (
    <>
      {transcodeVideos.size > 0 && failedVideos.size === 0 && (
        <div className="alert alert-info mx-3 mt-3 mb-0" role="alert">
          <small>
            {transcodeVideos.size} video{transcodeVideos.size > 1 ? 's are' : ' is'} being transcoded for browser compatibility...
          </small>
        </div>
      )}
      {failedVideos.size > 0 && (
        <div className="alert alert-warning mx-3 mt-3 mb-0" role="alert">
          <small>
            {failedVideos.size} video{failedVideos.size > 1 ? 's' : ''} could not be transcoded.
            These videos can still be viewed in the regular message view.
          </small>
        </div>
      )}
      <div className="media-grid">
        {displayableItems.map((item, index) => {
          const isVideo = item.media_type?.startsWith('video/')
          return (
            <div
              key={item.id}
              className="media-grid-item"
              onClick={() => handleThumbnailClick(index)}
            >
              <div className="media-thumbnail">
                {isVideo ? (
                  <>
                    <video
                      src={`${API_BASE}/media?id=${item.id}${transcodeVideos.has(item.id) ? '&transcode=true' : ''}#t=0.1`}
                      preload="metadata"
                      muted
                      playsInline
                      key={`${item.id}-${transcodeVideos.has(item.id)}`}
                      style={{
                        width: '100%',
                        height: '100%',
                        objectFit: 'cover',
                        display: 'block',
                        pointerEvents: 'none'
                      }}
                      onError={(e) => {
                        if (e.target.error?.code === 4) {
                          // MEDIA_ERR_SRC_NOT_SUPPORTED - unsupported format/codec
                          handleVideoError(item.id, transcodeVideos.has(item.id))
                        }
                      }}
                    />
                    <div className="video-indicator">
                      <svg
                        style={{ width: '2rem', height: '2rem' }}
                        fill="white"
                        viewBox="0 0 24 24"
                      >
                        <path d="M8 5v14l11-7z" />
                      </svg>
                    </div>
                  </>
                ) : (
                  <img
                    src={`${API_BASE}/media?id=${item.id}`}
                    alt={`Media ${index + 1}`}
                    loading="lazy"
                  />
                )}
              </div>
            </div>
          )
        })}
      </div>

      {selectedIndex !== null && (
        <MediaCarousel
          mediaItems={displayableItems}
          initialIndex={selectedIndex}
          onClose={handleCloseCarousel}
          transcodeVideos={transcodeVideos}
        />
      )}
    </>
  )
}

export default MediaGrid
