import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import EquipmentDisplay from '../components/EquipmentDisplay'
import PlayerHeader from '../components/PlayerHeader'
import { resourceAPI, userAPI } from '../lib/api'
import { useAuth } from '../context/AuthContext'
import './ResourcesGather.css'

const GATHER_COOLDOWN_SECONDS = 10
const GATHER_PROCESS_CACHE_KEY = 'moonshine:resources_gather_process'

const normalizeImagePath = (img) => {
  if (!img) return null
  let p = img.trim()
  if (p.startsWith('/')) p = p.slice(1)
  p = p.replace(/^frontend\/assets\/images\//, '')
  p = p.replace(/^assets\/images\//, '')
  if (!p.startsWith('images/')) p = `images/${p}`
  return `/assets/${p}`
}

export default function ResourcesGather() {
  const { location_slug } = useParams()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { logout, refetchUser } = useAuth()

  const [user, setUser] = useState(null)
  const [equippedItems, setEquippedItems] = useState({})
  const [resources, setResources] = useState([])
  const [selectedSlug, setSelectedSlug] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [gathering, setGathering] = useState(false)
  const [gatherCooldown, setGatherCooldown] = useState(0)
  const [gatherResultMessage, setGatherResultMessage] = useState('')
  const gatherCooldownTimerRef = useRef(null)
  const gatherResultTimerRef = useRef(null)

  const requestedSlug = searchParams.get('slug') || ''

  useEffect(() => {
    if (!location_slug) {
      setError('Location slug is required')
      setLoading(false)
      return
    }

    let cancelled = false

    setLoading(true)
    setError('')

    Promise.all([
      userAPI.getCurrentUser(),
      userAPI.getEquippedItems(),
      resourceAPI.getResources(location_slug),
    ])
      .then(([userData, equippedData, resourcesData]) => {
        if (cancelled) return

        setUser(userData)
        setEquippedItems(equippedData || {})
        setResources(resourcesData || [])

        if ((resourcesData || []).length === 0) {
          setSelectedSlug('')
          return
        }

        const hasRequested = resourcesData.some((resource) => resource.slug === requestedSlug)
        const nextSlug = hasRequested ? requestedSlug : resourcesData[0].slug
        setSelectedSlug(nextSlug)

        if (!hasRequested || !requestedSlug) {
          setSearchParams({ slug: nextSlug }, { replace: true })
        }
      })
      .catch((err) => {
        if (cancelled) return
        console.error('[ResourcesGather] Error loading page data:', err)

        if (err.message.includes('Unauthorized')) {
          localStorage.removeItem('token')
          navigate('/signin', { replace: true })
          return
        }

        setError(err.message || 'Ошибка загрузки ресурсов')
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [location_slug, requestedSlug, navigate, setSearchParams])

  const selectedResource = useMemo(() => {
    if (!resources.length) return null
    return resources.find((resource) => resource.slug === selectedSlug) || resources[0]
  }, [resources, selectedSlug])

  const handleLogout = () => {
    logout()
    localStorage.clear()
    navigate('/signin')
  }

  const handleBack = () => {
    if (!location_slug) {
      navigate('/locations/wayward_pines')
      return
    }

    if (location_slug.endsWith('cell')) {
      navigate('/locations/wayward_pines')
      return
    }

    navigate(`/locations/${location_slug}`)
  }

  const handleSelectResource = (slug) => {
    if (gathering || gatherCooldown > 0 || !slug || slug === selectedSlug) return
    setSelectedSlug(slug)
    setSearchParams({ slug }, { replace: true })
  }

  const persistGatherProcess = (resourceName) => {
    const payload = {
      locationSlug: location_slug || '',
      resourceName,
    }
    localStorage.setItem(GATHER_PROCESS_CACHE_KEY, JSON.stringify(payload))
  }

  const clearGatherProcess = () => {
    localStorage.removeItem(GATHER_PROCESS_CACHE_KEY)
  }

  const finishGatherProcess = (resourceName) => {
    clearGatherProcess()
    setGatherResultMessage(`Вы удачно собрали - ${resourceName}`)
    void refetchUser()
  }

  const startGatherCooldown = () => {
    if (gatherCooldownTimerRef.current) {
      window.clearInterval(gatherCooldownTimerRef.current)
      gatherCooldownTimerRef.current = null
    }

    let seconds = GATHER_COOLDOWN_SECONDS
    setGatherCooldown(seconds)

    gatherCooldownTimerRef.current = window.setInterval(() => {
      seconds -= 1
      if (seconds <= 0) {
        setGatherCooldown(0)
        window.clearInterval(gatherCooldownTimerRef.current)
        gatherCooldownTimerRef.current = null
        return
      }
      setGatherCooldown(seconds)
    }, 1000)
  }

  const startGatherFinalization = (resourceName) => {
    if (gatherResultTimerRef.current) {
      window.clearTimeout(gatherResultTimerRef.current)
      gatherResultTimerRef.current = null
    }

    gatherResultTimerRef.current = window.setTimeout(() => {
      finishGatherProcess(resourceName)
      gatherResultTimerRef.current = null
    }, GATHER_COOLDOWN_SECONDS * 1000)
  }

  const handleGather = async () => {
    if (!selectedResource || gathering || gatherCooldown > 0) return

    const resourceSlug = selectedResource.slug
    const resourceName = selectedResource.name
    setGatherResultMessage('')
    setGathering(true)

    try {
      await resourceAPI.gather(resourceSlug)
      persistGatherProcess(resourceName)
      startGatherCooldown()
      startGatherFinalization(resourceName)
    } catch (err) {
      console.error('[ResourcesGather] Error gathering resource:', err)
      clearGatherProcess()
      alert(err.message || 'Ошибка при сборе ресурса')
    } finally {
      setGathering(false)
    }
  }

  useEffect(() => {
    const cachedRaw = localStorage.getItem(GATHER_PROCESS_CACHE_KEY)
    if (!cachedRaw) {
      return
    }

    try {
      const cached = JSON.parse(cachedRaw)
      if (!cached || typeof cached !== 'object') {
        clearGatherProcess()
        return
      }

      if (cached.locationSlug && location_slug && cached.locationSlug !== location_slug) {
        return
      }

      const cachedResourceName = typeof cached.resourceName === 'string' && cached.resourceName.trim()
        ? cached.resourceName
        : 'ресурс'

      setGatherResultMessage('')
      startGatherCooldown()
      startGatherFinalization(cachedResourceName)
    } catch {
      clearGatherProcess()
    }
  }, [location_slug])

  useEffect(() => {
    return () => {
      if (gatherCooldownTimerRef.current) {
        window.clearInterval(gatherCooldownTimerRef.current)
      }
      if (gatherResultTimerRef.current) {
        window.clearTimeout(gatherResultTimerRef.current)
      }
    }
  }, [])

  if (loading) {
    return (
      <div className="resources-gather-container">
        <div className="resources-gather-main">
          <div className="resources-gather-content">
            <p>Загрузка...</p>
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="resources-gather-container">
        <div className="resources-gather-main">
          <div className="resources-gather-content">
            <p className="resources-gather-error">{error}</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="resources-gather-container">
      <div className="resources-gather-main">
        <div className="resources-gather-header">
          <PlayerHeader user={user} />
          <div className="resources-gather-header-actions">
            <button type="button" className="resources-gather-back" onClick={handleBack}>
              Назад
            </button>
            <button type="button" onClick={handleLogout} className="resources-gather-logout">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M3 21V3h8v2H5v14h6v2H3zm13-4l-1.375-1.45 2.55-2.55H9v-2h8.175l-2.55-2.55L16 7l5 5-5 5z" fill="currentColor" />
              </svg>
            </button>
          </div>
        </div>

        <div className="resources-gather-content">
          <div className="resources-gather-player">
            <EquipmentDisplay user={user} equippedItems={equippedItems} readonly={true} />
          </div>

          <div className="resources-gather-center">
            <div className="resources-gather-title">Сбор ресурсов</div>
            {resources.length > 1 ? (
              <div className="resources-gather-list">
                {resources.map((resource) => (
                  <button
                    key={resource.id}
                    type="button"
                    className={`resources-gather-item ${selectedResource?.slug === resource.slug ? 'active' : ''}`}
                    onClick={() => handleSelectResource(resource.slug)}
                    disabled={gathering || gatherCooldown > 0}
                  >
                    {resource.name}
                  </button>
                ))}
              </div>
            ) : null}
            <button
              type="button"
              className="resources-gather-action"
              onClick={handleGather}
              disabled={!selectedResource || gathering || gatherCooldown > 0}
            >
              {gathering ? 'Сбор...' : gatherCooldown > 0 ? `Идет сбор (${gatherCooldown}с)` : 'Собрать'}
            </button>
            {gatherResultMessage ? (
              <div className="resources-gather-success">{gatherResultMessage}</div>
            ) : null}
          </div>

          <div className="resources-gather-target">
            {selectedResource ? (
              <>
                <img
                  src={normalizeImagePath(selectedResource.image)}
                  alt={selectedResource.name}
                  className="resources-gather-image"
                />
                <div className="resources-gather-name">{selectedResource.name}</div>
              </>
            ) : (
              <div className="resources-gather-empty">Ресурсы не найдены</div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
