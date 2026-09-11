import { useState } from 'react'
import axios from 'axios'
import { Modal, Button, Form, Alert, Spinner, ProgressBar } from 'react-bootstrap'

const API_BASE = import.meta.env.VITE_API_URL || '/api'

function Upload({ onClose, onSuccess }) {
  const [files, setFiles] = useState([])
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState(null)
  const [success, setSuccess] = useState(null)
  const [progress, setProgress] = useState(null)
  const [uploadProgress, setUploadProgress] = useState(0)
  const [currentStep, setCurrentStep] = useState(1) // 1 = upload, 2 = processing
  const [uploadPhase, setUploadPhase] = useState(null) // step 1 sub-phase: 'sending' | 'saving'
  const [currentFileIndex, setCurrentFileIndex] = useState(0)
  const [totalFiles, setTotalFiles] = useState(0)
  const [isDragging, setIsDragging] = useState(false)

  const handleFileChange = (e) => {
    const selectedFiles = Array.from(e.target.files)
    setFiles(selectedFiles)
    setError(null)
    setSuccess(null)
  }

  const handleDragEnter = (e) => {
    e.preventDefault()
    e.stopPropagation()
    if (!uploading) {
      setIsDragging(true)
    }
  }

  const handleDragLeave = (e) => {
    e.preventDefault()
    e.stopPropagation()
    // Only set dragging to false if we're leaving the drop zone itself
    if (e.currentTarget === e.target) {
      setIsDragging(false)
    }
  }

  const handleDragOver = (e) => {
    e.preventDefault()
    e.stopPropagation()
  }

  const handleDrop = (e) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(false)

    if (uploading) return

    const droppedFiles = Array.from(e.dataTransfer.files)
    // Filter to only accept XML files
    const xmlFiles = droppedFiles.filter(file => file.name.toLowerCase().endsWith('.xml'))

    if (xmlFiles.length === 0) {
      setError('Please drop only XML files')
      return
    }

    if (xmlFiles.length < droppedFiles.length) {
      setError(`Only ${xmlFiles.length} of ${droppedFiles.length} files are XML files. Non-XML files were ignored.`)
    }

    setFiles(xmlFiles)
    setSuccess(null)
    if (xmlFiles.length === droppedFiles.length) {
      setError(null)
    }
  }

  const handleUpload = async () => {
    if (files.length === 0) {
      setError('Please select at least one file')
      return
    }

    setUploading(true)
    setError(null)
    setProgress(null)
    setTotalFiles(files.length)
    setCurrentFileIndex(0)

    try {
      // Process each file sequentially
      for (let i = 0; i < files.length; i++) {
        const file = files[i]
        setCurrentFileIndex(i + 1)

        await uploadSingleFile(file)

        // If this was the last file, show success and close
        if (i === files.length - 1) {
          setSuccess(`Successfully imported all ${files.length} file${files.length !== 1 ? 's' : ''}`)
          setTimeout(() => {
            setUploading(false)
            onSuccess()
          }, 1500)
        }
      }
    } catch (err) {
      console.error('Upload error:', err)
      setError(err.response?.data?.error || err.message || 'Upload failed')
      setUploading(false)
    }
  }

  const uploadSingleFile = async (file) => {
    setUploadProgress(0)
    setCurrentStep(1)
    setUploadPhase('sending')

    // Step 1a: Send the file to the server.
    //
    // We send the raw File as the request body via XMLHttpRequest, instead
    // of wrapping it in FormData. FormData forces the browser to serialize
    // the whole multipart body (boundary + file bytes) before XHR can send
    // a single byte, which on a multi-GB file is exactly what caused the
    // long "hangs at the start with no progress" behavior. Sending the File
    // directly as the body lets the browser stream it from disk as it
    // sends, while still going through XHR (not fetch()) so we get a real
    // upload.onprogress callback -- fetch() has no equivalent: its
    // ReadableStream bodies are drained into fetch()'s own internal buffer
    // as fast as the source can produce chunks, with no visibility into
    // how much has actually gone out over the wire.
    //
    // Note that upload.onprogress only reflects bytes handed to the OS
    // socket, not bytes the *server* has actually read off the wire and
    // saved to disk -- on a fast connection those can diverge a lot for a
    // large file, so this alone would still reach 100% before the server is
    // actually done. Step 1b (below) polls the server's own receive
    // progress to cover that gap.
    //
    // The two progress sources are independent and uncoordinated, and can
    // race and briefly disagree. Route both through this helper so the
    // displayed number never jumps backward or flickers between stale and
    // fresher reads.
    let displayedProgress = 0
    const reportProgress = (percent) => {
      displayedProgress = Math.max(displayedProgress, percent)
      setUploadProgress(displayedProgress)
    }

    const uploadPromise = new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      xhr.open('POST', `${API_BASE}/upload`)
      xhr.withCredentials = true
      xhr.setRequestHeader('Content-Type', 'application/octet-stream')
      xhr.setRequestHeader('X-Filename', file.name)

      // Browsers can fire upload.onprogress many times per second, far
      // faster than the UI needs -- throttle how often it actually
      // triggers a re-render so the percentage doesn't visibly flicker.
      let lastReported = 0
      xhr.upload.onprogress = (event) => {
        if (!event.lengthComputable) return
        const now = performance.now()
        const percent = Math.round((event.loaded / event.total) * 100)
        if (percent === 100 || now - lastReported >= 250) {
          lastReported = now
          reportProgress(percent)
        }
      }

      xhr.onload = () => {
        let parsed
        try {
          parsed = JSON.parse(xhr.responseText)
        } catch {
          reject(new Error(`Upload failed with status ${xhr.status}`))
          return
        }
        if (xhr.status < 200 || xhr.status >= 300 || !parsed.success) {
          reject(new Error(parsed.error || `Upload failed with status ${xhr.status}`))
          return
        }
        resolve(parsed)
      }

      xhr.onerror = () => reject(new Error('Upload failed'))
      xhr.onabort = () => reject(new Error('Upload cancelled'))

      xhr.send(file)
    })

    // Step 1b: Track the server's receive/save progress in parallel with
    // the send. The server starts reading the request body as soon as it
    // arrives (before the whole thing is sent, and definitely before it's
    // fully written to disk for a multi-GB file), so this can legitimately
    // start showing progress while the XHR send is still in flight, and is
    // what carries the bar the rest of the way once the send itself hits
    // 100%.
    let pollReceiveProgress = true
    const receiveProgressPromise = (async () => {
      while (pollReceiveProgress) {
        await new Promise(r => setTimeout(r, 250))
        try {
          const response = await axios.get(`${API_BASE}/progress`)
          const data = response.data
          if (data?.status === 'receiving' && data.total_bytes > 0) {
            setUploadPhase('saving')
            reportProgress(Math.min(100, Math.round((data.bytes_received / data.total_bytes) * 100)))
          } else if (data?.status && data.status !== 'no_upload') {
            // Server has moved past receiving (e.g. into parsing) --
            // nothing more for this poll loop to do.
            return
          }
        } catch (err) {
          console.error('Error checking receive progress:', err)
        }
      }
    })()

    try {
      await uploadPromise
    } finally {
      pollReceiveProgress = false
      await receiveProgressPromise
    }

    setUploadProgress(100)

    // File uploaded successfully, move to step 2
    setUploadProgress(0) // Reset for processing step
    setUploadPhase(null)
    setCurrentStep(2)

    // Wait for processing to complete
    await waitForProcessingComplete()
  }

  const waitForProcessingComplete = () => {
    return new Promise((resolve, reject) => {
      const checkProgress = setInterval(async () => {
        try {
          const response = await axios.get(`${API_BASE}/progress`)
          const data = response.data

          if (!data || data.status === 'no_upload') {
            clearInterval(checkProgress)
            // Reject instead of resolve so the error is caught by handleUpload
            reject(new Error('Processing status unavailable'))
            return
          }

          setProgress(data)

          // Calculate processing progress (0-100% for step 2)
          const total = data.total_messages || 1
          const processed = data.processed_messages || 0
          const processingPercent = Math.min(Math.round((processed / total) * 100), 100)
          setUploadProgress(processingPercent)

          // Check if completed
          if (data.status === 'completed') {
            clearInterval(checkProgress)
            setUploadProgress(100)
            // Just resolve - don't call onSuccess() here since we're processing multiple files
            // The main handleUpload() function will handle success after all files are done
            resolve()
          } else if (data.status === 'error') {
            clearInterval(checkProgress)
            // Reject instead of resolve so the error is caught by handleUpload
            reject(new Error(data.error_message || 'Processing failed'))
          }
        } catch (err) {
          console.error('Error checking progress:', err)
        }
      }, 500) // Check every 500ms for more responsive updates
    })
  }

  return (
    <Modal show={true} onHide={onClose} centered backdrop="static" keyboard={!uploading}>
      <Modal.Header closeButton={!uploading}>
        <Modal.Title className="h4 fw-bold">Upload Backup</Modal.Title>
      </Modal.Header>

      <Modal.Body>
        <div className="mb-3">
          <div className="d-flex align-items-center gap-2 text-muted mb-3">
            <svg style={{width: '1.25rem', height: '1.25rem'}} className="text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <small>Select or drag and drop one or more XML files from SMS Backup & Restore app</small>
          </div>

          <Form.Group>
            <div
              onDragEnter={handleDragEnter}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onDrop={handleDrop}
              style={{
                border: isDragging ? '2px dashed #0d6efd' : '2px dashed #dee2e6',
                borderRadius: '0.375rem',
                padding: '2rem 1rem',
                textAlign: 'center',
                backgroundColor: isDragging ? '#f0f7ff' : '#f8f9fa',
                transition: 'all 0.2s ease',
                cursor: uploading ? 'not-allowed' : 'pointer',
                opacity: uploading ? 0.6 : 1
              }}
            >
              <div className="mb-3">
                <svg
                  style={{width: '3rem', height: '3rem'}}
                  className={isDragging ? "text-primary" : "text-muted"}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                </svg>
              </div>
              <div className={isDragging ? "text-primary fw-semibold" : "text-muted"}>
                {isDragging ? (
                  <div>Drop XML files here</div>
                ) : (
                  <div>
                    <div className="mb-2">Drag and drop XML files here</div>
                    <div className="text-muted small">or</div>
                  </div>
                )}
              </div>
              <div className="mt-3">
                <Form.Control
                  type="file"
                  id="backupFileInput"
                  name="backupFileInput"
                  aria-label="Select backup XML files"
                  accept=".xml"
                  onChange={handleFileChange}
                  disabled={uploading}
                  multiple
                  style={{
                    maxWidth: '250px',
                    margin: '0 auto'
                  }}
                />
              </div>
            </div>
            {files.length > 0 && !uploading && (
              <div className="mt-3">
                <Form.Text className="text-success d-flex align-items-center gap-1">
                  <svg style={{width: '1rem', height: '1rem'}} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  {files.length} file{files.length !== 1 ? 's' : ''} selected
                </Form.Text>
                <div className="mt-2" style={{maxHeight: '150px', overflowY: 'auto'}}>
                  {files.map((file, index) => (
                    <div key={index} className="small text-muted">
                      {index + 1}. {file.name} ({(file.size / (1024 * 1024)).toFixed(2)} MB)
                    </div>
                  ))}
                </div>
              </div>
            )}
          </Form.Group>
        </div>

        {uploading && (
          <div className="mb-3">
            {totalFiles > 1 && (
              <div className="mb-2">
                <small className="text-muted fw-semibold">
                  Processing file {currentFileIndex} of {totalFiles}
                </small>
              </div>
            )}
            <div className="d-flex justify-content-between align-items-center mb-2">
              <small className="text-muted fw-semibold">
                Step {currentStep} of 2: {currentStep === 1
                  ? (uploadPhase === 'saving' ? 'Saving file on server' : 'Uploading file')
                  : 'Processing messages'}
              </small>
              <small className="text-muted fw-bold">{uploadProgress}%</small>
            </div>
            <ProgressBar
              now={uploadProgress}
              variant={uploadProgress === 100 && currentStep === 2 ? "success" : "primary"}
              striped={!(uploadProgress === 100 && currentStep === 2)}
              animated={!(uploadProgress === 100 && currentStep === 2)}
            />
            {currentStep === 1 && files[currentFileIndex - 1] && (
              <small className="text-muted mt-2 d-block">
                {uploadPhase === 'saving'
                  ? `Server is saving ${files[currentFileIndex - 1].name} (${(files[currentFileIndex - 1].size / (1024 * 1024)).toFixed(2)} MB)...`
                  : `Uploading ${files[currentFileIndex - 1].name} (${(files[currentFileIndex - 1].size / (1024 * 1024)).toFixed(2)} MB) to server...`}
              </small>
            )}
            {currentStep === 2 && progress && (
              <small className="text-muted mt-2 d-block">
                {progress.processed_messages?.toLocaleString() || 0} / {progress.total_messages?.toLocaleString() || '?'} messages imported
                {progress.processed_calls > 0 && `, ${progress.processed_calls?.toLocaleString()} calls`}
              </small>
            )}
            {currentStep === 2 && !progress && (
              <small className="text-muted mt-2 d-block">
                Starting import process...
              </small>
            )}
          </div>
        )}

        {error && (
          <Alert variant="danger" className="d-flex align-items-center gap-2">
            <svg style={{width: '1.25rem', height: '1.25rem'}} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            {error}
          </Alert>
        )}

        {success && (
          <Alert variant="success" className="d-flex align-items-center gap-2">
            <svg style={{width: '1.25rem', height: '1.25rem'}} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            {success}
          </Alert>
        )}
      </Modal.Body>

      <Modal.Footer>
        <Button variant="secondary" onClick={onClose} disabled={uploading}>
          Cancel
        </Button>
        <Button variant="primary" onClick={handleUpload} disabled={uploading || files.length === 0}>
          {uploading ? (
            <>
              <Spinner
                as="span"
                animation="border"
                size="sm"
                role="status"
                aria-hidden="true"
                className="me-2"
              />
              Uploading...
            </>
          ) : 'Upload'}
        </Button>
      </Modal.Footer>
    </Modal>
  )
}

export default Upload
