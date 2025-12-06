import { useState, useEffect, useRef } from 'react'
import { format } from 'date-fns'
import './MediaCarousel.css'

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8081/api'

function MediaCarousel({ mediaItems, initialIndex, onClose, transcodeVideos = new Set() }) {
  const [currentIndex, setCurrentIndex] = useState(initialIndex)
  const [touchStart, setTouchStart] = useState(null)
  const [touchEnd, setTouchEnd] = useState(null)
  const videoRef = useRef(null)

  const currentItem = mediaItems[currentIndex]
  const isVideo = currentItem?.media_type?.startsWith('video/')
  const shouldTranscode = transcodeVideos.has(currentItem?.id)

  useEffect(() => {
    // Prevent body scroll when carousel is open
    document.body.style.overflow = 'hidden'

    // Hide UI elements that might appear over the carousel
    const datePickers = document.querySelectorAll('.react-datepicker-popper')
    datePickers.forEach(picker => {
      picker.style.display = 'none'
    })

    const dateFilterContainer = document.querySelector('.date-filter-container')
    if (dateFilterContainer) {
      dateFilterContainer.style.visibility = 'hidden'
    }

    const header = document.querySelector('header')
    if (header) {
      header.style.visibility = 'hidden'
    }

    return () => {
      document.body.style.overflow = 'unset'

      // Restore visibility of hidden elements
      const datePickers = document.querySelectorAll('.react-datepicker-popper')
      datePickers.forEach(picker => {
        picker.style.display = ''
      })

      const dateFilterContainer = document.querySelector('.date-filter-container')
      if (dateFilterContainer) {
        dateFilterContainer.style.visibility = ''
      }

      const header = document.querySelector('header')
      if (header) {
        header.style.visibility = ''
      }
    }
  }, [])

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === 'Escape') {
        onClose()
      } else if (e.key === 'ArrowLeft') {
        handlePrevious()
      } else if (e.key === 'ArrowRight') {
        handleNext()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [currentIndex])

  const handlePrevious = () => {
    if (currentIndex > 0) {
      setCurrentIndex(currentIndex - 1)
    }
  }

  const handleNext = () => {
    if (currentIndex < mediaItems.length - 1) {
      setCurrentIndex(currentIndex + 1)
    }
  }

  // Touch handlers for swipe gestures
  const handleTouchStart = (e) => {
    setTouchEnd(null)
    setTouchStart(e.targetTouches[0].clientX)
  }

  const handleTouchMove = (e) => {
    setTouchEnd(e.targetTouches[0].clientX)
  }

  const handleTouchEnd = () => {
    if (!touchStart || !touchEnd) return

    const distance = touchStart - touchEnd
    const isLeftSwipe = distance > 50
    const isRightSwipe = distance < -50

    if (isLeftSwipe) {
      handleNext()
    }
    if (isRightSwipe) {
      handlePrevious()
    }
  }

  const formatTime = (date) => {
    return format(new Date(date), 'MMM d, yyyy h:mm a')
  }

  return (
    <div
      className="media-carousel"
      onClick={onClose}
      onTouchStart={handleTouchStart}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
    >
      {/* Close button */}
      <button
        className="carousel-close-btn"
        onClick={onClose}
        aria-label="Close"
      >
        <svg style={{ width: '1.5rem', height: '1.5rem' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>

      {/* Counter */}
      <div className="carousel-counter">
        {currentIndex + 1} / {mediaItems.length}
      </div>

      {/* Media info */}
      <div className="carousel-info">
        <div>{formatTime(currentItem.date)}</div>
        {currentItem.body && (
          <div className="text-muted small mt-1">{currentItem.body}</div>
        )}
      </div>

      {/* Previous button */}
      {currentIndex > 0 && (
        <button
          className="carousel-nav-btn carousel-prev-btn"
          onClick={(e) => {
            e.stopPropagation()
            handlePrevious()
          }}
          aria-label="Previous"
        >
          <svg style={{ width: '2rem', height: '2rem' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>
      )}

      {/* Next button */}
      {currentIndex < mediaItems.length - 1 && (
        <button
          className="carousel-nav-btn carousel-next-btn"
          onClick={(e) => {
            e.stopPropagation()
            handleNext()
          }}
          aria-label="Next"
        >
          <svg style={{ width: '2rem', height: '2rem' }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </button>
      )}

      {/* Media content */}
      <div
        className="carousel-media-container"
        onClick={(e) => e.stopPropagation()}
      >
        {isVideo ? (
          <video
            ref={videoRef}
            controls
            autoPlay
            playsInline
            className="carousel-media"
            src={`${API_BASE}/media?id=${currentItem.id}${shouldTranscode ? '&transcode=true' : ''}`}
            key={`${currentItem.id}-${shouldTranscode}`}
            onError={(e) => {
              console.error('Video playback error:', e)
              console.error('Video src:', `${API_BASE}/media?id=${currentItem.id}${shouldTranscode ? '&transcode=true' : ''}`)
              console.error('Video error code:', e.target.error?.code)
              console.error('Video error message:', e.target.error?.message)
            }}
            onLoadStart={() => console.log('Video load started:', currentItem.id)}
            onCanPlay={() => console.log('Video can play:', currentItem.id)}
          />
        ) : (
          <img
            src={`${API_BASE}/media?id=${currentItem.id}`}
            alt={`Media ${currentIndex + 1}`}
            className="carousel-media"
          />
        )}
      </div>
    </div>
  )
}

export default MediaCarousel
