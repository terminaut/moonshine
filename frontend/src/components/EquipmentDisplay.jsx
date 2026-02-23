import './EquipmentDisplay.css'
import StatsDisplay from './StatsDisplay'

export default function EquipmentDisplay({ user, equippedItems, onSlotClick, readonly = false, avatarCacheBuster = '' }) {
  const normalizeImagePath = (img) => {
    if (!img) return null
    let p = img.trim()
    if (p.startsWith('/')) p = p.slice(1)
    p = p.replace(/^frontend\/assets\/images\//, '')
    p = p.replace(/^assets\/images\//, '')
    if (p.startsWith('images/')) {
      p = p.replace(/^images\//, '')
    }
    if (p.includes('bots/') || p.includes('rat') || p.includes('bear') || p.includes('spider') || p.includes('skeleton') || p.includes('zombie') || p.includes('boar')) {
      if (!p.includes('bots/')) {
        p = `bots/${p}`
      }
      if (!p.includes('.')) {
        p = `${p}.jpg`
      }
    }
    const normalizedPath = `/assets/images/${p}`
    if (avatarCacheBuster && p.startsWith('players/avatars/')) {
      return `${normalizedPath}?v=${avatarCacheBuster}`
    }
    return normalizedPath
  }

  const getEquipmentSlotImage = (slotName) => {
    const equippedItem = equippedItems?.[slotName] || equippedItems?.[slotName.toLowerCase()]
    if (equippedItem && equippedItem.image) {
      return normalizeImagePath(equippedItem.image)
    }
    const placeholderName = slotName.startsWith('ring') ? 'ring' : slotName
    return `/assets/images/equipment_items/grid/${placeholderName}.png`
  }

  const hasEquippedItem = (slotName) => {
    const equippedItem = equippedItems?.[slotName] || equippedItems?.[slotName.toLowerCase()]
    return equippedItem && equippedItem.image
  }

  const getGridSlotImage = (slotName) => `/assets/images/equipment_items/grid/${slotName}.png`

  const renderEquipmentSlot = (slotName, alt) => {
    const hasItem = hasEquippedItem(slotName)
    const clickable = hasItem && !readonly && onSlotClick

    return (
      <div
        className={`equipment-slot ${hasItem ? 'has-item' : ''} ${readonly ? 'readonly' : ''}`}
        onClick={() => clickable && onSlotClick(slotName)}
        style={{ cursor: clickable ? 'pointer' : 'default' }}
        title={clickable ? 'Нажмите чтобы снять' : ''}
      >
        <img src={getEquipmentSlotImage(slotName)} alt={alt} />
      </div>
    )
  }

  const renderStaticSlot = (slotName, alt, extraClass = '') => (
    <div className={`equipment-slot readonly ${extraClass}`.trim()}>
      <img src={getGridSlotImage(slotName)} alt={alt} />
    </div>
  )

  const avatarImage = user?.avatar
  const avatarSrc = avatarImage ? normalizeImagePath(avatarImage) : getEquipmentSlotImage('head')

  return (
    <div className="equipment-display-wrapper">
      <div className="equipment-grid">
        <div className="equipment-column-left">
          {renderEquipmentSlot('head', 'head')}
          {renderEquipmentSlot('neck', 'neck')}
          {renderEquipmentSlot('weapon', 'weapon')}
          {renderEquipmentSlot('legs', 'legs')}
          {renderEquipmentSlot('feet', 'feet')}
        </div>

        <div className="equipment-column-center">
          <div className="equipment-avatar">
            {avatarSrc ? (
              <img
                src={avatarSrc}
                alt={user?.username || user?.name || 'Avatar'}
                onError={(e) => {
                  e.target.src = getEquipmentSlotImage('head')
                }}
              />
            ) : (
              <img src={getEquipmentSlotImage('head')} alt="avatar placeholder" />
            )}
          </div>

          <StatsDisplay
            attack={user?.attack}
            defense={user?.defense}
            hp={user?.hp}
            currentHp={user?.currentHp ?? user?.current_hp}
            gold={user?.gold}
            showGold={true}
          />
        </div>

        <div className="equipment-column-right">
          <div className="equipment-row-top">
            {renderStaticSlot('bag', 'bag', 'equipment-bag-offset-right')}
            {renderStaticSlot('bag', 'bag', 'equipment-bag-offset-right')}
            {renderStaticSlot('bag', 'bag')}
          </div>
          <div className="equipment-column-right-items">
            {renderEquipmentSlot('arms', 'arms')}
            {renderEquipmentSlot('hands', 'hands')}
            {renderEquipmentSlot('shield', 'shield')}
            {renderEquipmentSlot('chest', 'chest')}
            {renderEquipmentSlot('belt', 'belt')}
            <div className="equipment-double-slot-row">
              {renderEquipmentSlot('ring1', 'ring1')}
              {renderEquipmentSlot('ring2', 'ring2')}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
