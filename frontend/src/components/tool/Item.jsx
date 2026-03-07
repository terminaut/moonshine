export default function ToolItem({ item }) {
  if (!item) return null

  return (
    <div className="row top">
      <div className="col-md-3">
        <img src={item.image?.url} alt={item.name} />
      </div>

      <div className="col-md-9">
        <h3>{item.name}</h3>
      </div>
    </div>
  )
}
