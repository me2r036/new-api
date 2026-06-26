/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { type SVGProps } from 'react'
import { cn } from '@/lib/utils'

export function IconGoogle({ className, ...props }: SVGProps<SVGSVGElement>) {
  return (
    <svg
      role='img'
      viewBox='0 0 24 24'
      xmlns='http://www.w3.org/2000/svg'
      width='24'
      height='24'
      className={cn(className)}
      {...props}
    >
      <title>Google</title>
      <path
        d='M21.805 12.233c0-.747-.067-1.465-.191-2.154H12v4.077h5.498a4.704 4.704 0 0 1-2.04 3.087v2.563h3.303c1.933-1.779 3.044-4.403 3.044-7.573Z'
        fill='#4285f4'
      />
      <path
        d='M12 22c2.754 0 5.063-.913 6.75-2.194l-3.303-2.563c-.913.612-2.082.972-3.447.972-2.648 0-4.891-1.788-5.693-4.192H2.893v2.644A9.997 9.997 0 0 0 12 22Z'
        fill='#34a853'
      />
      <path
        d='M6.307 14.023A5.998 5.998 0 0 1 5.989 12c0-.703.121-1.384.318-2.023V7.333H2.893A9.997 9.997 0 0 0 2 12c0 1.613.386 3.14.893 4.667l3.414-2.644Z'
        fill='#fbbc04'
      />
      <path
        d='M12 5.785c1.498 0 2.844.515 3.904 1.527l2.927-2.927C17.058 2.743 14.749 2 12 2a9.997 9.997 0 0 0-9.107 5.333l3.414 2.644C7.109 7.573 9.352 5.785 12 5.785Z'
        fill='#ea4335'
      />
    </svg>
  )
}
