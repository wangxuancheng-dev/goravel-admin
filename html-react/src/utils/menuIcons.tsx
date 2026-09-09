import {
  ApartmentOutlined,
  AppstoreOutlined,
  BankOutlined,
  BellOutlined,
  BookOutlined,
  ClockCircleOutlined,
  CloudServerOutlined,
  CodeOutlined,
  DashboardOutlined,
  FileTextOutlined,
  FolderOutlined,
  FormOutlined,
  IdcardOutlined,
  LockOutlined,
  MenuOutlined,
  MonitorOutlined,
  PaperClipOutlined,
  PayCircleOutlined,
  SettingOutlined,
  ShopOutlined,
  TeamOutlined,
  ToolOutlined,
  UserOutlined,
  WalletOutlined,
} from '@ant-design/icons'
import type { ComponentType } from 'react'

/** Map Element Plus-style menu icon names (from backend) to Ant Design icons. */
const ICON_MAP: Record<string, ComponentType> = {
  Setting: SettingOutlined,
  User: UserOutlined,
  UserFilled: TeamOutlined,
  Lock: LockOutlined,
  Menu: MenuOutlined,
  OfficeBuilding: BankOutlined,
  Postcard: IdcardOutlined,
  Monitor: MonitorOutlined,
  Timer: ClockCircleOutlined,
  Collection: BookOutlined,
  Tools: ToolOutlined,
  Download: PaperClipOutlined,
  Paperclip: PaperClipOutlined,
  CircleClose: LockOutlined,
  ShoppingCart: ShopOutlined,
  Wallet: WalletOutlined,
  CreditCard: PayCircleOutlined,
  Coin: PayCircleOutlined,
  Avatar: UserOutlined,
  Document: FileTextOutlined,
  DocumentCopy: FileTextOutlined,
  Tickets: FileTextOutlined,
  Notebook: FileTextOutlined,
  DataLine: CloudServerOutlined,
  Odometer: DashboardOutlined,
  Bell: BellOutlined,
  Cpu: CodeOutlined,
  Edit: FormOutlined,
  Folder: FolderOutlined,
  Grid: AppstoreOutlined,
  Connection: ApartmentOutlined,
}

export function resolveMenuIcon(iconName?: string) {
  if (!iconName) return undefined
  const Comp = ICON_MAP[iconName] || AppstoreOutlined
  return <Comp />
}
