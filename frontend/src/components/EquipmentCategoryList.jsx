import { Link } from 'react-router-dom'
import './EquipmentCategoryList.css'

const categories = [
  { slug: 'weapon', filename: '1-weapon' },
  { slug: 'head', filename: '2-head' },
  { slug: 'chest', filename: '3-chest' },
  { slug: 'legs', filename: '4-legs' },
  { slug: 'feet', filename: '5-feet' },
  { slug: 'arms', filename: '6-arms' },
  { slug: 'hands', filename: '7-hands' },
  { slug: 'belt', filename: '8-belt' },
  { slug: 'ring', filename: '9-ring' },
  { slug: 'shield', filename: '10-shield' },
]

export default function EquipmentCategoryList({ currentCategory, artifact = false }) {
  return (
    <div className="equipment-category-list">
      {categories.map((category) => (
        <Link
          key={category.slug}
          to={`/equipment_items?category=${category.slug}&artifact=${artifact ? 'true' : 'false'}`}
          className={`equipment-category-item ${currentCategory === category.slug ? 'active' : ''}`}
        >
          <img
            src={`/assets/images/locations/cities/moonshine/weapon_shop/categories/${category.filename}.png`}
            alt={category.slug}
            className="equipment-category-icon"
          />
        </Link>
      ))}
    </div>
  )
}

