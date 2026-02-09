import type { UserRole } from '../types';

/**
 * Role Hierarchy (highest to lowest):
 * 1. sysadmin - Full system access, audit logs, system configuration
 * 2. admin - System configuration, leave types, holidays
 * 3. hr - User management, all leave requests, reports
 * 4. hod - Head of Department, approves team leave requests
 * 5. manager - Manages direct reports, approves team leave requests
 * 6. staff - Basic user, can only manage their own leave requests
 */

// Role hierarchy levels
const ROLE_LEVELS: Record<UserRole, number> = {
    sysadmin: 100,
    admin: 80,
    hr: 60,
    hod: 40,
    manager: 30,
    staff: 10,
};

/**
 * Check if user has at least the specified role level
 */
export const hasRoleLevel = (userRole: UserRole | undefined, minRole: UserRole): boolean => {
    if (!userRole) return false;
    return ROLE_LEVELS[userRole] >= ROLE_LEVELS[minRole];
};

/**
 * Check if user can manage team (approve/reject leave requests)
 * Includes: manager, hod, hr, admin, sysadmin
 */
export const canManageTeam = (role?: UserRole): boolean =>
    role === 'manager' || role === 'hod' || role === 'hr' || role === 'admin' || role === 'sysadmin';

/**
 * Check if user has HR permissions
 * Includes: hr, admin, sysadmin
 */
export const hasHRPermissions = (role?: UserRole): boolean =>
    role === 'hr' || role === 'admin' || role === 'sysadmin';

/**
 * Check if user has admin permissions
 * Includes: admin, sysadmin
 */
export const hasAdminPermissions = (role?: UserRole): boolean =>
    role === 'admin' || role === 'sysadmin';

/**
 * Check if user is system administrator
 */
export const isSysAdmin = (role?: UserRole): boolean =>
    role === 'sysadmin';

/**
 * Get human-readable role label
 */
export const getRoleLabel = (role?: UserRole): string => {
    switch (role) {
        case 'sysadmin': return 'System Administrator';
        case 'admin': return 'Administrator';
        case 'hr': return 'Human Resources';
        case 'hod': return 'Head of Department';
        case 'manager': return 'Manager';
        case 'staff': return 'Staff';
        default: return role || 'Unknown';
    }
};

/**
 * Get role badge color class
 */
export const getRoleBadgeColor = (role?: UserRole): string => {
    switch (role) {
        case 'sysadmin': return 'bg-red-100 text-red-700';
        case 'admin': return 'bg-purple-100 text-purple-700';
        case 'hr': return 'bg-blue-100 text-blue-700';
        case 'hod': return 'bg-amber-100 text-amber-700';
        case 'manager': return 'bg-teal-100 text-teal-700';
        case 'staff': return 'bg-slate-100 text-slate-700';
        default: return 'bg-slate-100 text-slate-700';
    }
};

/**
 * Get all roles that can be assigned by the given role
 * - sysadmin can assign any role
 * - admin can assign hr, hod, manager, staff
 * - hr can only assign staff (when creating users)
 */
export const getAssignableRoles = (currentUserRole?: UserRole): UserRole[] => {
    switch (currentUserRole) {
        case 'sysadmin':
            return ['sysadmin', 'admin', 'hr', 'hod', 'manager', 'staff'];
        case 'admin':
            return ['hr', 'hod', 'manager', 'staff'];
        case 'hr':
            return ['hod', 'manager', 'staff'];
        default:
            return [];
    }
};

/**
 * Check if workflow step is actionable by user's role
 */
export const canPerformWorkflowAction = (userRole?: UserRole, stepRole?: UserRole): boolean => {
    if (!userRole || !stepRole) return false;

    // User can perform action if their role matches or they have higher permissions
    if (userRole === stepRole) return true;

    // Admin and sysadmin can perform any action
    if (userRole === 'admin' || userRole === 'sysadmin') return true;

    // HR can perform HR and below actions
    if (userRole === 'hr' && (stepRole === 'hr' || stepRole === 'hod' || stepRole === 'manager' || stepRole === 'staff')) return true;

    return false;
};
