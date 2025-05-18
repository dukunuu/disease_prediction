import { Link, NavLink } from "react-router"

const NAVBAR_ROUTES = [
  {name: 'Өвчтөн', to: '/patients'},
  {name: 'Бүртгэх', to: '/get-disease'}
]

export default function Navbar() {
  return (
    <nav className="w-full flex py-5 justify-between px-10 dark:bg-white/10 bg-black/20">
      <Link to="/" className="text-xl uppercase font-bold">
        Онош тодорхойлох
      </Link>
      <ul className="flex gap-5">
        {NAVBAR_ROUTES.map(route => (
          <li key={route.name}>
            <NavLink to={route.to} className={({isActive}) => `${isActive && 'text-blue-500'} hover:text-blue-400 transition-colors`}>{route.name}</NavLink>
          </li>
        ))}
      </ul>
    </nav>
  )
};
