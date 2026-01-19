export interface User {
    canLogin: boolean;
    isAdmin: boolean;
    createdAt: string;
    deletedAt: string | null;
    email: string;
    firstName: string | null;
    id: number;
    lastName: string | null;
    updatedAt: string | null;
}
