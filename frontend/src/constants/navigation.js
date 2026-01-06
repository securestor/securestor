import {
  Database,
  Package,
  Shield,
  FileText,
  Users,
  CheckSquare,
  Key,
  UserCheck,
  Building2,
  Settings,
  RefreshCw,
  Activity,
  LayoutDashboard,
  HardDrive,
} from "lucide-react";
import FEATURES from '../config/features';

// Base navigation items (always available)
const BASE_NAV_ITEMS = [
  { id: "overview", icon: LayoutDashboard, label: "Overview" },
  { id: "repositories", icon: Database, label: "Repositories" },
  { id: "artifacts", icon: Package, label: "Artifacts" },
  { id: "security", icon: Shield, label: "Security Scan" },
  { id: "logs", icon: FileText, label: "Logs" },
  { id: "api-keys", icon: Key, label: "API Keys" },
];

// Enterprise-only navigation items
const ENTERPRISE_NAV_ITEMS = [
  { id: "compliance", icon: CheckSquare, label: "Compliance", feature: 'COMPLIANCE_MANAGEMENT' },
  { id: "users", icon: Users, label: "User Management", feature: 'USER_MANAGEMENT' },
  { id: "roles", icon: UserCheck, label: "Role Management", feature: 'ROLE_MANAGEMENT' },
  { id: "tenant-management", icon: Building2, label: "Tenant Management", feature: 'TENANT_MANAGEMENT' },
  { id: "cache", icon: HardDrive, label: "Cache Management", feature: 'CACHE_MANAGEMENT' },
  { id: "tenant-settings", icon: Settings, label: "Tenant Settings", feature: 'TENANT_SETTINGS' },
  { id: "replication-settings", icon: RefreshCw, label: "Replication Settings", feature: 'REPLICATION' },
];

// Build navigation items based on enabled features
export const NAV_ITEMS = [
  ...BASE_NAV_ITEMS,
  ...ENTERPRISE_NAV_ITEMS.filter(item => FEATURES[item.feature])
];
