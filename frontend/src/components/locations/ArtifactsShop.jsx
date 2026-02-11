import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { preloadImages } from '../../lib/imageCache'
import './WeaponShop.css'

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

export default function ArtifactsShop() {
  useEffect(() => {
    preloadImages([
      '/assets/images/locations/cities/moonshine/shop_of_artifacts/bg.jpg',
      ...categories.map((category) => `/assets/images/locations/cities/moonshine/weapon_shop/categories/${category.filename}.png`),
    ])
  }, [])

  return (
    <div className="weapon-shop-container">
      <div className="weapon-shop-bg">
        <img
          src="/assets/images/locations/cities/moonshine/shop_of_artifacts/bg.jpg"
          alt="Артефакты"
          className="weapon-shop-bg-image"
          decoding="async"
        />
      </div>
      <div className="weapon-shop-categories">
        {categories.map((category) => (
          <Link
            key={category.slug}
            to={`/equipment_items?category=${category.slug}&artifact=true`}
            className="weapon-shop-category-link"
          >
            <img
              src={`/assets/images/locations/cities/moonshine/weapon_shop/categories/${category.filename}.png`}
              alt={category.slug}
              className="weapon-shop-category-icon"
              decoding="async"
            />
          </Link>
        ))}
      </div>
    </div>
  )
}
