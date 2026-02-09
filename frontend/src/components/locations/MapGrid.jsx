import { useEffect, useState } from 'react'
import { useAuth } from '../../context/AuthContext'
import { locationAPI } from '../../lib/api'
import './MapGrid.css'

export default function MapGrid({ locationSlug }) {
  const { user, refetchUser } = useAuth()
  const [cells, setCells] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [moving, setMoving] = useState(false)
  const [movementInfo, setMovementInfo] = useState(null)
  const [remainingTime, setRemainingTime] = useState(0)

  useEffect(() => {
    const loadCells = async () => {
      try {
        setLoading(true)
        const data = await locationAPI.getCells(locationSlug)
        setCells(data.cells || [])
        setError(null)
      } catch (err) {
        console.error('[MapGrid] Error loading cells:', err)
        setError(err.message)
      } finally {
        setLoading(false)
      }
    }

    if (locationSlug) {
      loadCells()
    }
  }, [locationSlug])

  useEffect(() => {
    if (!locationSlug) return

    const intervalId = setInterval(() => {
      refetchUser()
    }, 2000)

    return () => clearInterval(intervalId)
  }, [locationSlug, refetchUser])

  const handleCellClick = async (e, cellSlug) => {
    e.preventDefault()
    
    if (moving || user?.locationSlug === cellSlug) {
      return
    }

    try {
      setMoving(true)
      const response = await locationAPI.moveToCell(locationSlug, cellSlug)
      
      if (response && response.path_length > 0) {
        const totalTime = response.path_length * (response.time_per_cell || 5)
        const targetName = (response.target_cell || cellSlug).replace(/cell$/, '')
        setRemainingTime(totalTime)
        setMovementInfo({
          targetCell: targetName,
          totalTime
        })
      }
      
      await refetchUser()
    } catch (err) {
      console.error('[MapGrid] Error moving to cell:', err)
      alert(err.message || 'Ошибка при перемещении')
    } finally {
      setMoving(false)
    }
  }

  useEffect(() => {
    if (!movementInfo) return

    const intervalId = setInterval(() => {
      setRemainingTime((prev) => {
        if (prev <= 1) {
          setMovementInfo(null)
          refetchUser()
          return 0
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(intervalId)
  }, [movementInfo, refetchUser])

  if (loading) {
    return <div className="map-grid-loading">Загрузка карты...</div>
  }

  if (error) {
    return <div className="map-grid-error">Ошибка загрузки: {error}</div>
  }

  const gridSize = 8
  const totalCells = gridSize * gridSize
  const currentSlug = user?.locationSlug || user?.location?.slug || ''
  const playerCellSlug = locationSlug === 'wayward_pines'
    ? (currentSlug.endsWith('cell') ? currentSlug : '29cell')
    : currentSlug
  const cellMap = new Map()
  cells.forEach((cell) => {
    if (!cell || !cell.slug) {
      return
    }
    const match = cell.slug.match(/^(\d+)cell$/)
    if (match) {
      const cellNum = parseInt(match[1], 10)
      cellMap.set(cellNum, cell)
    }
  })

  return (
    <div className="map-grid">
      {movementInfo && (
        <div className="map-movement-indicator">
          Переход на клетку: <strong>{movementInfo.targetCell}</strong>, осталось: <strong>{remainingTime}с</strong>
        </div>
      )}
      <div className="map-grid-container">
        {Array.from({ length: totalCells }, (_, index) => {
          const cellNum = index + 1
          const cell = cellMap.get(cellNum)
          const isPlayerHere = playerCellSlug === cell?.slug

          if (!cell) {
            return (
              <div key={index} className="map-cell map-cell-empty" />
            )
          }

          return (
            <div key={cell.id} className="map-cell-wrapper">
              <div
                onClick={(e) => handleCellClick(e, cell.slug)}
                className={`map-cell ${cell.inactive ? 'map-cell-inactive' : ''} ${moving ? 'map-cell-moving' : ''}`}
                title={cell.name}
                style={{ cursor: moving ? 'wait' : 'pointer' }}
              >
                {cell.image && (
                  <img
                    src={`/assets/images/locations/${cell.image}`}
                    alt={cell.name}
                    className="map-cell-image"
                  />
                )}
              </div>
              {isPlayerHere && (
                <img
                  src="/assets/images/warrior.png"
                  alt="Персонаж"
                  className="map-cell-player-icon"
                />
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
