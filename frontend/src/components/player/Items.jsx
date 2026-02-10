import { useState } from 'react'
import EquipmentItem from '../equipment/Item'
import ToolItem from '../tool/Item'
import ResourceItem from '../resources/Item'
import config from '../../config'

export default function Items({ player }) {
  const [currentCategory, setCurrentCategory] = useState('all')
  const [errorMessage, setErrorMessage] = useState('')
  const [sellingItemId, setSellingItemId] = useState(null)
  const [sellingResourceId, setSellingResourceId] = useState(null)

  if (!player) return null

  const cleanErrors = () => {
    setErrorMessage('')
    setSellingItemId(null)
    setSellingResourceId(null)
  }

  const putOnItem = async (item, type) => {
    try {
      const response = await fetch(`${config.apiUrl}/stuff/items/${item.id}/put_on`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
        body: JSON.stringify({ item_type: type }),
      })
      
      if (!response.ok) {
        console.error('Failed to put on item')
        return
      }
      
      const data = await response.json()
    } catch (error) {
      console.error('Error putting on item:', error)
    }
  }

  const sellItem = async (item, type) => {
    cleanErrors()

    try {
      const response = await fetch(`${config.apiUrl}/stuff/items/${item.id}/sell`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
        body: JSON.stringify({ item_type: type }),
      })
      
      if (!response.ok) {
        const error = await response.text()
        setErrorMessage(error)
        if (type === 'resource') {
          setSellingResourceId(item.id)
        } else {
          setSellingItemId(item.id)
        }
        return
      }
      
      const data = await response.json()
    } catch (error) {
      setErrorMessage('Failed to sell item')
      if (type === 'resource') {
        setSellingResourceId(item.id)
      } else {
        setSellingItemId(item.id)
      }
    }
  }

  return (
    <div>
      <nav className="navbar">
        <ul>
          <li>
            <a onClick={() => setCurrentCategory('all')} className="btn btn-default">All</a>
          </li>
          <li>
            <a onClick={() => setCurrentCategory('equipment_items')} className="btn btn-default">Equipment</a>
          </li>
          <li>
            <a onClick={() => setCurrentCategory('tool_items')} className="btn btn-default">Tools</a>
          </li>
          <li>
            <a onClick={() => setCurrentCategory('resources')} className="btn btn-default">Resources</a>
          </li>
        </ul>
      </nav>

      <div className="items">
        {(currentCategory === 'all' || currentCategory === 'equipment_items') && (
          <div className="equipment_items">
            {player.equipment_items?.length > 0 ? (
              player.equipment_items.map((item) => (
                <div key={item.id} className="item row">
                  <EquipmentItem item={item} playerLevel={player.level} />
                  <a onClick={() => putOnItem(item, 'equipment')} className="btn btn-info">Put on</a>
                  <a onClick={() => sellItem(item, 'equipment')} className="btn btn-info">
                    Sell for {item.sell_price} gold
                  </a>
                  {errorMessage && sellingItemId === item.id && (
                    <div className="alert alert-danger" role="alert">
                      {errorMessage}
                    </div>
                  )}
                </div>
              ))
            ) : (
              <h2>No equipment items</h2>
            )}
          </div>
        )}

        {(currentCategory === 'all' || currentCategory === 'tool_items') && (
          <div className="tool_items">
            {player.tool_items?.length > 0 ? (
              player.tool_items.map((item) => {
                const categoryName = item.category?.name?.toLowerCase()
                const playerSkill = player[`${categoryName}_skill`] || 0
                return (
                  <div key={item.id} className="item row">
                    <ToolItem item={item} playerSkill={playerSkill} />
                    <a onClick={() => putOnItem(item, 'tool')} className="btn btn-info">Put on</a>
                    <a onClick={() => sellItem(item, 'tool')} className="btn btn-info">
                      Sell for {item.sell_price} gold
                    </a>
                    {errorMessage && sellingItemId === item.id && (
                      <div className="alert alert-danger" role="alert">
                        {errorMessage}
                      </div>
                    )}
                  </div>
                )
              })
            ) : (
              <h2>No tool items</h2>
            )}
          </div>
        )}

        {(currentCategory === 'all' || currentCategory === 'resources') && (
          <div className="resources">
            {player.resources?.length > 0 ? (
              player.resources.map((item) => (
                <div key={item.id} className="item row">
                  <ResourceItem item={item} />
                  <a onClick={() => sellItem(item, 'resource')} className="btn btn-info">
                    Sell for {item.price} gold
                  </a>
                  {errorMessage && sellingResourceId === item.id && (
                    <div className="alert alert-danger" role="alert">
                      {errorMessage}
                    </div>
                  )}
                </div>
              ))
            ) : (
              <h2>No resources</h2>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
















