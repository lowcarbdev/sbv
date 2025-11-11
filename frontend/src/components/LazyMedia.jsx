import { useState, useEffect, useRef } from 'react'
import axios from 'axios'
import VCardPreview from './VCardPreview'

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8081/api'

function LazyMedia({ messageId, mediaType, className, alt = "MMS attachment" }) {
  const [src, setSrc] = useState(null)
  const [vcfData, setVcfData] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(false)
  const [showModal, setShowModal] = useState(false)
  const imgRef = useRef(null)
  const videoRef = useRef(null)
  const observerRef = useRef(null)
  const hasLoadedRef = useRef(false)

  useEffect(() => {
    // Reset loaded state when messageId changes
    hasLoadedRef.current = false

    // Set up Intersection Observer for lazy loading
    observerRef.current = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting && !hasLoadedRef.current) {
            hasLoadedRef.current = true
            loadMedia()
          }
        })
      },
      {
        // Only load images below viewport (not above) to prevent scroll jump
        // rootMargin: top right bottom left
        rootMargin: '50px 0px 200px 0px'
      }
    )

    if (imgRef.current) {
      observerRef.current.observe(imgRef.current)
    }

    return () => {
      if (observerRef.current) {
        observerRef.current.disconnect()
      }
    }
  }, [messageId])

  const loadMedia = async () => {
    setLoading(true)
    try {
      // Check if this is a VCF file - fetch as text instead of blob
      const isVCard = mediaType === 'text/x-vcard' ||
                      mediaType === 'text/vcard' ||
                      mediaType === 'text/directory'

      if (isVCard) {
        // Fetch VCF as text
        const response = await axios.get(`${API_BASE}/media`, {
          params: { id: messageId },
          responseType: 'text'
        })
        setVcfData(response.data)
      } else {
        // Fetch other media as blob
        const response = await axios.get(`${API_BASE}/media`, {
          params: { id: messageId },
          responseType: 'blob'
        })

        const blob = response.data
        const objectUrl = URL.createObjectURL(blob)
        setSrc(objectUrl)
      }

      // Stop observing once loaded - we don't need to track this element anymore
      if (observerRef.current && imgRef.current) {
        observerRef.current.unobserve(imgRef.current)
      }
    } catch (err) {
      console.error('Failed to load media:', err)
      setError(true)
    } finally {
      setLoading(false)
    }
  }

  // Cleanup object URL when component unmounts
  useEffect(() => {
    return () => {
      if (src) {
        URL.revokeObjectURL(src)
      }
    }
  }, [src])

  // Handle ESC key to close modal
  useEffect(() => {
    const handleEscape = (e) => {
      if (e.key === 'Escape' && showModal) {
        setShowModal(false)
      }
    }

    if (showModal) {
      document.addEventListener('keydown', handleEscape)
      // Prevent body scroll when modal is open
      document.body.style.overflow = 'hidden'
    }

    return () => {
      document.removeEventListener('keydown', handleEscape)
      document.body.style.overflow = 'unset'
    }
  }, [showModal])

  // Pause original video when modal opens
  useEffect(() => {
    if (showModal && videoRef.current) {
      videoRef.current.pause()
    }
  }, [showModal])

  if (!mediaType) {
    return null
  }

  const isImage = mediaType.startsWith('image/')
  const isVideo = mediaType.startsWith('video/')
  const isVCard = mediaType === 'text/x-vcard' ||
                  mediaType === 'text/vcard' ||
                  mediaType === 'text/directory'

  return (
    <>
      <div ref={imgRef} className={className}>
      {/* Placeholder shown before loading or while loading */}
      {!src && !vcfData && !error && (
        <div
          className="bg-light rounded d-flex align-items-center justify-content-center position-relative overflow-hidden"
          style={{
            width: '100%',
            aspectRatio: isVideo ? '16/9' : '3/4', // Common phone camera ratio
            minHeight: isVideo ? '200px' : '300px', // Larger to prevent layout shift
            maxHeight: '400px',
            backgroundColor: '#f8f9fa',
            backgroundImage: 'linear-gradient(45deg, #e9ecef 25%, transparent 25%, transparent 75%, #e9ecef 75%, #e9ecef), linear-gradient(45deg, #e9ecef 25%, transparent 25%, transparent 75%, #e9ecef 75%, #e9ecef)',
            backgroundSize: '20px 20px',
            backgroundPosition: '0 0, 10px 10px'
          }}
        >
          <div className="text-center">
            {loading ? (
              <>
                <div className="spinner-border spinner-border-sm text-secondary mb-2" role="status">
                  <span className="visually-hidden">Loading...</span>
                </div>
                <div className="small text-muted">Loading {isImage ? 'image' : isVideo ? 'video' : isVCard ? 'contact' : 'media'}...</div>
              </>
            ) : (
              <div className="text-muted d-flex flex-column align-items-center">
                {isImage && (
                  <svg style={{width: '2.5rem', height: '2.5rem'}} className="mb-2 text-secondary opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                )}
                {isVideo && (
                  <svg style={{width: '2.5rem', height: '2.5rem'}} className="mb-2 text-secondary opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                )}
                {isVCard && (
                  <svg style={{width: '2.5rem', height: '2.5rem'}} className="mb-2 text-secondary opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                  </svg>
                )}
                {!isImage && !isVideo && !isVCard && (
                  <svg style={{width: '2.5rem', height: '2.5rem'}} className="mb-2 text-secondary opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                  </svg>
                )}
                <small className="text-muted">
                  {isImage ? 'Image' : isVideo ? 'Video' : isVCard ? 'Contact' : 'Attachment'}
                </small>
              </div>
            )}
          </div>
        </div>
      )}

      {error && (
        <div className="alert alert-warning mb-0 small">
          <div className="d-flex align-items-center gap-2">
            <svg style={{width: '1.25rem', height: '1.25rem'}} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            Failed to load media
          </div>
        </div>
      )}

      {(src || vcfData) && !loading && !error && (
        <>
          {isImage && src && (
            <img
              src={src}
              alt={alt}
              className="img-fluid rounded shadow"
              loading="lazy"
              onClick={() => setShowModal(true)}
              style={{
                display: 'block',
                maxWidth: '100%',
                maxHeight: '400px',
                objectFit: 'contain',
                animation: 'fadeIn 0.3s ease-in',
                cursor: 'pointer'
              }}
            />
          )}
          {isVideo && src && (
            <video
              ref={videoRef}
              controls
              className="img-fluid rounded shadow"
              src={src}
              onClick={(e) => {
                e.preventDefault()
                if (videoRef.current) {
                  videoRef.current.pause()
                }
                setShowModal(true)
              }}
              style={{
                display: 'block',
                maxWidth: '100%',
                maxHeight: '400px',
                objectFit: 'contain',
                animation: 'fadeIn 0.3s ease-in',
                cursor: 'pointer'
              }}
            />
          )}
          {isVCard && vcfData && (
            <VCardPreview vcfText={vcfData} messageId={messageId} />
          )}
          {!isImage && !isVideo && !isVCard && (
            <div className="small p-2 rounded bg-light d-flex align-items-center gap-1">
              <svg style={{width: '1rem', height: '1rem'}} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
              </svg>
              Attachment: {mediaType}
            </div>
          )}
        </>
      )}
      </div>

      {/* Full-screen modal */}
      {showModal && (isImage || isVideo) && src && (
        <div
          className="position-fixed top-0 start-0 w-100 h-100 d-flex align-items-center justify-content-center"
          style={{
            backgroundColor: 'rgba(0, 0, 0, 0.9)',
            zIndex: 9999,
            padding: '2rem'
          }}
          onClick={() => setShowModal(false)}
        >
          {/* Close button */}
          <button
            className="btn btn-light position-absolute top-0 end-0 m-3"
            onClick={() => setShowModal(false)}
            style={{
              zIndex: 10000,
              borderRadius: '50%',
              width: '40px',
              height: '40px',
              padding: 0,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center'
            }}
          >
            <svg style={{width: '1.5rem', height: '1.5rem'}} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>

          {/* Media content - stop propagation to prevent closing when clicking on media */}
          <div
            onClick={(e) => e.stopPropagation()}
            className="d-flex align-items-center justify-content-center"
            style={{
              maxWidth: '95vw',
              maxHeight: '95vh'
            }}
          >
            {isImage && (
              <img
                src={src}
                alt={alt}
                className="rounded shadow-lg"
                style={{
                  maxWidth: '100%',
                  maxHeight: '95vh',
                  objectFit: 'contain'
                }}
              />
            )}
            {isVideo && (
              <video
                controls
                autoPlay
                className="rounded shadow-lg"
                src={src}
                style={{
                  maxWidth: '100%',
                  maxHeight: '95vh',
                  objectFit: 'contain'
                }}
              />
            )}
          </div>
        </div>
      )}
    </>
  )
}

export default LazyMedia
