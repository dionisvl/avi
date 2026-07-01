const MODES = Object.freeze({
    LOGIN: 'login',
    REGISTER: 'register',
    VERIFY: 'verify',
    DASHBOARD: 'dashboard',
    ITEMS: 'items',
    ITEM_DETAIL: 'item-detail',
    ITEM_CREATE: 'item-create',
    ITEM_EDIT: 'item-edit',
    FAVORITES: 'favorites',
    PROFILE: 'profile',
    CHAT: 'chat',
    CONTACT: 'contact',
});

const EMPTY_PAGINATION = { page: 1, per_page: 20, total: 0, total_pages: 1 };

const emptyLogin = () => ({ email: '', password: '' });
const emptyRegister = () => ({ email: '', password: '', passwordConfirm: '' });
const emptyVerify = () => ({ email: '', code: '' });
const emptyContact = () => ({ name: '', email: '', subject: '', message: '' });
const emptyProfilePreferences = () => ({
    category_id: '',
    city_id: '',
    condition: '',
    price_min: '',
    price_max: '',
    search: '',
});

const emptyItemFilters = () => ({
    search: '',
    category_id: '',
    condition: '',
    city_uuid: '',
    price_min: '',
    price_max: '',
    status: '',
    seller_id: '',
    mine_only: false,
});

const emptyItemForm = () => ({
    title: '',
    description: '',
    category_id: '',
    condition: 'used',
    city_uuid: '',
    price_amount: '',
    price_currency: 'RUB',
    status: 'published',
    tags: '',
});

function localizedName(entity, locale = 'en') {
    if (!entity) return '';
    if (typeof entity.name === 'string') return entity.name;
    const names = entity.names || entity.localized_name || {};
    return names[locale] || names.en || names.ru || entity.slug || '';
}

function normalizeItem(item, locale = 'en') {
    if (!item) return null;

    return {
        ...item,
        title: item.title || item.name || '',
        photos: Array.isArray(item.photos) ? item.photos : [],
        tags: Array.isArray(item.tags) ? item.tags : [],
        category_name: localizedName(item.category, locale),
        city_name: localizedName(item.city, locale),
        seller: item.seller || null,
        is_favorited: item.is_favorited ?? false,
    };
}

function formatPrice(price) {
    if (!price || price.amount == null || !price.currency) return '-';
    const amount = Number(price.amount);
    const currency = String(price.currency).toUpperCase();
    if (!Number.isFinite(amount)) return '-';

    try {
        return new Intl.NumberFormat('en', { style: 'currency', currency }).format(amount / 100);
    } catch (error) {
        return `${amount} ${currency}`;
    }
}

function buildStorageURL(apiUrl, path) {
    if (!path) return '';
    try {
        return new URL(path, apiUrl).toString();
    } catch (error) {
        return path;
    }
}

export function createApp() {
    return {
        MODES,
        envLoaded: false,
        apiUrl: '',
        locale: typeof localStorage !== 'undefined' ? (localStorage.getItem('ui_locale') || 'en') : 'en',
        currentToken: typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null,
        mode: typeof localStorage !== 'undefined' && localStorage.getItem('auth_token') ? MODES.DASHBOARD : MODES.LOGIN,
        loading: false,
        errorMessage: '',
        successMessage: '',
        me: {},
        cities: [],
        categories: [],
        loginForm: emptyLogin(),
        registerForm: emptyRegister(),
        verifyForm: emptyVerify(),
        pendingRegistrationEmail: '',
        items: [],
        itemsPagination: EMPTY_PAGINATION,
        itemsLoading: false,
        itemFilters: emptyItemFilters(),
        itemDetail: null,
        itemForm: emptyItemForm(),
        itemEditId: null,
        itemReturnMode: MODES.ITEMS,
        itemFormError: '',
        itemFormLoading: false,
        itemPhotoIds: [],
        itemPhotoPond: null,
        favorites: [],
        favoritesPagination: EMPTY_PAGINATION,
        favoritesLoading: false,
        contactForm: emptyContact(),
        contactLoading: false,
        contactStatus: '',
        contactError: '',
        profileForm: { name: '' },
        profilePreferencesForm: emptyProfilePreferences(),
        profileStatus: '',
        profileError: '',
        profileLoading: false,
        profileAvatarPond: null,
        avatarUploadStatus: '',
        avatarUploadError: false,
        changePasswordForm: { current_password: '', new_password: '' },
        changePasswordStatus: '',
        changePasswordError: '',
        changePasswordLoading: false,
        deleteAccountForm: { password: '' },
        deleteAccountError: '',
        deleteAccountLoading: false,
        promotionPaymentLoading: false,
        promotionPaymentError: '',
        promotionPaymentStatus: '',
        promotionPaymentAmount: null,
        promotionPaymentReturnItemID: '',
        chatConversations: [],
        chatConversationsLoading: false,
        chatConversationsError: '',
        chatSelectedConversation: null,
        chatMessages: [],
        chatMessagesLoading: false,
        chatMessagesError: '',
        chatMessageBody: '',
        chatSending: false,
        chatRealtimeTimer: null,
        authRedirectTimer: null,
        isApplyingRoute: false,
        routePopHandler: null,

        get isAuthenticated() {
            return Boolean(this.currentToken);
        },

        get hasAdminRole() {
            return Array.isArray(this.me?.roles) && this.me.roles.includes('ROLE_ADMIN');
        },

        isItemMode(mode = this.mode) {
            return [MODES.ITEMS, MODES.ITEM_DETAIL, MODES.ITEM_CREATE, MODES.ITEM_EDIT].includes(mode);
        },

        isItemFormMode(mode = this.mode) {
            return [MODES.ITEM_CREATE, MODES.ITEM_EDIT].includes(mode);
        },

        normalizeRoutePath(path = window.location.pathname) {
            const normalized = `/${String(path || '').split('?')[0].split('#')[0].split('/').filter(Boolean).join('/')}`;
            return normalized === '/' ? '/' : normalized.replace(/\/+$/, '');
        },

        normalizeRouteTarget(target = `${window.location.pathname}${window.location.search}`) {
            const url = new URL(target || '/', window.location.origin);
            return `${this.normalizeRoutePath(url.pathname)}${url.search}`;
        },

        parseBrowserRoute(path = `${window.location.pathname}${window.location.search}`) {
            const normalized = this.normalizeRoutePath(path);
            const parts = normalized.split('/').filter(Boolean);
            if (parts.length === 0) return { name: 'home', path: '/', protected: true };

            const [section, id, action] = parts;
            if (section === 'login') return { name: MODES.LOGIN, path: '/login', protected: false };
            if (section === 'register') return { name: MODES.REGISTER, path: '/register', protected: false };
            if (section === 'verify') return { name: MODES.VERIFY, path: '/verify', protected: false };
            if (section === 'items' && !id) return { name: MODES.ITEMS, path: '/items', protected: true };
            if (section === 'items' && id === 'new') return { name: MODES.ITEM_CREATE, path: '/items/new', protected: true };
            if (section === 'items' && id && action === 'edit') return { name: MODES.ITEM_EDIT, itemID: id, path: `/items/${id}/edit`, protected: true };
            if (section === 'items' && id) return { name: MODES.ITEM_DETAIL, itemID: id, path: `/items/${id}`, protected: true };
            if (section === 'favorites') return { name: MODES.FAVORITES, path: '/favorites', protected: true };
            if (section === 'profile') return { name: MODES.PROFILE, path: '/profile', protected: true };
            if (section === 'chat') return { name: MODES.CHAT, path: '/chat', protected: true };
            if (section === 'contact') return { name: MODES.CONTACT, path: '/contact', protected: true };
            return { name: 'not-found', path: normalized, protected: true };
        },

        routePathForMode(mode = this.mode) {
            if (mode === MODES.LOGIN) return '/login';
            if (mode === MODES.REGISTER) return '/register';
            if (mode === MODES.VERIFY) return '/verify';
            if (mode === MODES.ITEMS || mode === MODES.DASHBOARD) return this.itemsRoutePath();
            if (mode === MODES.ITEM_CREATE) return '/items/new';
            if (mode === MODES.ITEM_DETAIL && this.itemDetail?.id) return `/items/${this.itemDetail.id}`;
            if (mode === MODES.ITEM_EDIT && this.itemEditId) return `/items/${this.itemEditId}/edit`;
            if (mode === MODES.FAVORITES) return '/favorites';
            if (mode === MODES.PROFILE) return '/profile';
            if (mode === MODES.CHAT) return '/chat';
            if (mode === MODES.CONTACT) return '/contact';
            return null;
        },

        writeBrowserRoute(path, { replace = false } = {}) {
            if (this.isApplyingRoute || !path) return;
            const normalized = this.normalizeRouteTarget(path);
            const current = this.normalizeRouteTarget();
            if (current === normalized && !window.location.hash) return;

            const url = new URL(window.location.href);
            const next = new URL(normalized, window.location.origin);
            url.pathname = next.pathname;
            url.search = next.search;
            url.hash = '';
            window.history[replace ? 'replaceState' : 'pushState']({}, document.title, url);
        },

        syncBrowserRoute({ replace = false } = {}) {
            this.writeBrowserRoute(this.routePathForMode(), { replace });
        },

        authReturnPathFromURL() {
            const params = new URLSearchParams(window.location.search);
            return params.get('return_to') || '';
        },

        setAuthReturnPath(path) {
            const target = this.normalizeRouteTarget(path);
            if (!target || ['/login', '/register', '/verify'].includes(this.normalizeRoutePath(target))) return '';
            sessionStorage.setItem('auth_return_path', target);
            return target;
        },

        consumeAuthReturnPath() {
            const target = this.authReturnPathFromURL() || sessionStorage.getItem('auth_return_path') || '';
            sessionStorage.removeItem('auth_return_path');
            return target ? this.normalizeRouteTarget(target) : '';
        },

        loginRouteWithReturn(path) {
            const target = this.setAuthReturnPath(path);
            if (!target) return '/login';
            const query = new URLSearchParams({ return_to: target });
            return `/login?${query.toString()}`;
        },

        async applyBrowserRoute(path = `${window.location.pathname}${window.location.search}`, { replace = false } = {}) {
            const route = this.parseBrowserRoute(path);
            const hadAuth = this.isAuthenticated;
            let canonicalPath = route.path;

            this.isApplyingRoute = true;
            try {
                if (!this.isAuthenticated && route.protected) {
                    this.mode = MODES.LOGIN;
                    canonicalPath = this.loginRouteWithReturn(path);
                    return;
                }

                if (this.isAuthenticated && [MODES.LOGIN, MODES.REGISTER, MODES.VERIFY].includes(route.name)) {
                    await this.goToItems({ skipRoute: true });
                    canonicalPath = '/items';
                    return;
                }

                if (route.name === MODES.LOGIN || route.name === MODES.REGISTER || route.name === MODES.VERIFY) {
                    this.mode = route.name;
                    this.errorMessage = '';
                    const returnTo = route.name === MODES.LOGIN ? new URL(path, window.location.origin).searchParams.get('return_to') : '';
                    canonicalPath = returnTo ? this.loginRouteWithReturn(returnTo) : route.path;
                    return;
                }

                if (route.name === 'home' || route.name === 'not-found' || route.name === MODES.ITEMS) {
                    this.applyItemFiltersFromSearch(new URL(path, window.location.origin).search);
                    await this.goToItems({ skipRoute: true });
                    canonicalPath = this.itemsRoutePath();
                    return;
                }

                if (route.name === MODES.ITEM_CREATE) {
                    this.openItemCreate({ skipRoute: true });
                    return;
                }

                if (route.name === MODES.ITEM_DETAIL) {
                    await this.openItemDetail(route.itemID, { skipRoute: true });
                    if (!this.itemDetail?.id) canonicalPath = '/items';
                    return;
                }

                if (route.name === MODES.ITEM_EDIT) {
                    await this.openItemEditByID(route.itemID, { skipRoute: true });
                    if (this.mode !== MODES.ITEM_EDIT) canonicalPath = '/items';
                    return;
                }

                if (route.name === MODES.FAVORITES) {
                    await this.goToFavorites({ skipRoute: true });
                    return;
                }

                if (route.name === MODES.PROFILE) {
                    this.goToProfile({ skipRoute: true });
                    return;
                }

                if (route.name === MODES.CHAT) {
                    await this.goToChat({ skipRoute: true });
                    return;
                }

                if (route.name === MODES.CONTACT) {
                    this.goToContact({ skipRoute: true });
                }
            } finally {
                this.isApplyingRoute = false;
                if (hadAuth && !this.isAuthenticated && route.protected) canonicalPath = '/login';
                if (replace && canonicalPath) this.writeBrowserRoute(canonicalPath, { replace: true });
            }
        },

        async setup() {
            if (!window.ENV?.API_URL) throw new Error('API_URL is not configured.');
            this.apiUrl = `${window.ENV.API_URL}/api/v1`;
            this.envLoaded = true;
            this.routePopHandler = () => this.applyBrowserRoute(`${window.location.pathname}${window.location.search}`);
            window.addEventListener('popstate', this.routePopHandler);

            await Promise.allSettled([this.loadCities(), this.loadCategories()]);

            const params = new URLSearchParams(window.location.search);
            const email = params.get('email');
            const code = params.get('code');
            const paymentReturn = params.get('payment_return');
            const paymentReturnItemID = params.get('item_id');
            let routeHandled = false;

            if (email && code) {
                this.verifyForm.email = email;
                this.verifyForm.code = code;
                this.mode = MODES.VERIFY;
                this.writeBrowserRoute('/verify', { replace: true });
                routeHandled = true;
            }

            if (paymentReturn === 'promote_listing' && paymentReturnItemID) {
                this.promotionPaymentReturnItemID = paymentReturnItemID;
                this.promotionPaymentStatus = 'Payment page closed. Waiting for provider confirmation...';
            }

            if (this.isAuthenticated) {
                await this.loadMe();
                if (this.promotionPaymentReturnItemID) {
                    await this.openItemDetail(this.promotionPaymentReturnItemID, { replace: true });
                    routeHandled = true;
                }
            }

            if (!routeHandled) await this.applyBrowserRoute(`${window.location.pathname}${window.location.search}`, { replace: true });

            this.$watch('mode', (next, prev) => {
                if (prev === MODES.CHAT && next !== MODES.CHAT) this.stopChatPolling();
            });
        },

        switchMode(mode, { replace = false, skipRoute = false } = {}) {
            this.mode = mode;
            this.errorMessage = '';
            if (!skipRoute) this.syncBrowserRoute({ replace });
        },

        getErrorMessage(error, fallback = 'Something went wrong') {
            if (error?.authSessionExpired) return '';
            if (error instanceof Error && error.message) return error.message;
            return fallback;
        },

        createHttpError(status, data) {
            const error = new Error(data.detail || data.error || `Request failed (${status})`);
            error.status = status;
            error.payload = data;
            return error;
        },

        handleHttpError(status, error, { on401 = 'throw', on404 = 'throw' } = {}) {
            if (status === 401 && on401 === 'logout') {
                error.authSessionExpired = true;
                this.handleLogout({ enforceLoginRedirect: true });
            }
            if (status === 404 && on404 === 'ignore') {
                return;
            }
            throw error;
        },

        async request(url, { method = 'GET', body, auth = false, on401 = auth ? 'logout' : 'throw', on404 = 'throw', allowStatuses = [] } = {}) {
            const headers = {};
            if (body) headers['Content-Type'] = 'application/json';
            if (auth) headers.Authorization = `Bearer ${this.currentToken}`;

            const res = await fetch(url, {
                method,
                headers,
                ...(body && { body: JSON.stringify(body) }),
            });

            const text = await res.text();
            let data = {};
            if (text) {
                try {
                    data = JSON.parse(text);
                } catch (error) {
                    data = {};
                }
            }

            if (!res.ok && !allowStatuses.includes(res.status)) {
                this.handleHttpError(res.status, this.createHttpError(res.status, data), { on401, on404 });
            }

            return data;
        },

        async withLoading(fn) {
            this.loading = true;
            this.errorMessage = '';
            try {
                await fn();
            } catch (error) {
                this.errorMessage = this.getErrorMessage(error);
            } finally {
                this.loading = false;
            }
        },

        cityName(city) {
            return localizedName(city, this.locale);
        },

        categoryName(category) {
            return localizedName(category, this.locale);
        },

        activeCities() {
            return this.cities.filter((city) => city.is_active !== false);
        },

        formatPrice,

        async loadCities() {
            try {
                const data = await this.request(`${this.apiUrl}/cities`);
                this.cities = Array.isArray(data.data) ? data.data : [];
            } catch (error) {
                this.cities = [];
            }
        },

        async loadCategories() {
            try {
                const data = await this.request(`${this.apiUrl}/categories?locale=${this.locale}`);
                this.categories = Array.isArray(data.data) ? data.data : [];
            } catch (error) {
                this.categories = [];
            }
        },

        async setLocale(locale) {
            if (!['en', 'ru'].includes(locale) || this.locale === locale) return;
            this.locale = locale;
            localStorage.setItem('ui_locale', locale);
            await this.loadCategories();
            this.items = this.items.map((item) => normalizeItem(item, this.locale));
            this.favorites = this.favorites.map((favorite) => ({
                ...favorite,
                item: normalizeItem(favorite.item, this.locale),
            }));
            if (this.itemDetail) this.itemDetail = normalizeItem(this.itemDetail, this.locale);
        },

        async loadMe() {
            try {
                this.me = await this.request(`${this.apiUrl}/user/me`, { auth: true });
                if (this.mode === MODES.DASHBOARD) this.mode = MODES.ITEMS;
            } catch (error) {
                // request() handles auth expiry.
            }
        },

        async handleLogin() {
            const { email, password } = this.loginForm;
            if (!email || !password) {
                this.errorMessage = 'Please fill in all fields';
                return;
            }

            await this.withLoading(async () => {
                const data = await this.request(`${this.apiUrl}/auth/login`, {
                    method: 'POST',
                    body: { email, password },
                });
                if (!data.access_token) throw new Error('Invalid response from server');
                localStorage.setItem('auth_token', data.access_token);
                this.currentToken = data.access_token;
                this.loginForm.password = '';
                await this.loadMe();
                const returnPath = this.consumeAuthReturnPath();
                if (returnPath) {
                    await this.applyBrowserRoute(returnPath, { replace: true });
                } else {
                    await this.goToItems({ replace: true });
                }
            });
        },

        async handleRegister() {
            const { email, password, passwordConfirm } = this.registerForm;
            if (!email || !password || !passwordConfirm) {
                this.errorMessage = 'Please fill in all fields';
                return;
            }
            if (password !== passwordConfirm) {
                this.errorMessage = 'Passwords do not match';
                return;
            }

            await this.withLoading(async () => {
                await this.request(`${this.apiUrl}/auth/register`, {
                    method: 'POST',
                    body: { email, password, locale: 'ru' },
                });
                this.pendingRegistrationEmail = email;
                this.verifyForm.email = email;
                this.registerForm = emptyRegister();
                this.switchMode(MODES.VERIFY);
            });
        },

        async handleVerifyEmail() {
            const { email, code } = this.verifyForm;
            if (!email || !code) {
                this.errorMessage = 'Please enter verification code';
                return;
            }

            await this.withLoading(async () => {
                await this.request(`${this.apiUrl}/auth/verify-email`, {
                    method: 'POST',
                    body: { email, code },
                });
                this.loginForm.email = email;
                this.verifyForm = emptyVerify();
                this.switchMode(MODES.LOGIN, { replace: true });
                this.successMessage = 'Email verified. Please log in.';
            });
        },

        enforceLoggedOutLoginState() {
            if (this.currentToken) return;
            this.isApplyingRoute = false;
            Object.assign(this, {
                mode: MODES.LOGIN,
                loading: false,
                itemsLoading: false,
                itemFormLoading: false,
                favoritesLoading: false,
                profileLoading: false,
                changePasswordLoading: false,
                deleteAccountLoading: false,
                chatConversationsLoading: false,
                chatMessagesLoading: false,
                chatSending: false,
                errorMessage: '',
                itemFormError: '',
                profileError: '',
                changePasswordError: '',
                deleteAccountError: '',
                chatConversationsError: '',
                chatMessagesError: '',
            });
            this.writeBrowserRoute('/login', { replace: true });
        },

        scheduleLoginRedirectAfterLogout() {
            if (this.authRedirectTimer) window.clearTimeout(this.authRedirectTimer);
            this.authRedirectTimer = window.setTimeout(() => {
                this.authRedirectTimer = null;
                this.enforceLoggedOutLoginState();
            }, 0);
        },

        handleLogout({ enforceLoginRedirect = false } = {}) {
            localStorage.removeItem('auth_token');
            sessionStorage.removeItem('auth_return_path');
            this.stopChatPolling();
            if (this.itemPhotoPond) this.itemPhotoPond.destroy();
            if (this.profileAvatarPond) this.profileAvatarPond.destroy();

            Object.assign(this, {
                currentToken: null,
                me: {},
                mode: MODES.LOGIN,
                loginForm: emptyLogin(),
                registerForm: emptyRegister(),
                verifyForm: emptyVerify(),
                items: [],
                itemFilters: emptyItemFilters(),
                itemDetail: null,
                itemForm: emptyItemForm(),
                favorites: [],
                errorMessage: '',
                successMessage: '',
                itemPhotoPond: null,
                profileAvatarPond: null,
            });
            this.writeBrowserRoute('/login', { replace: true });
            if (enforceLoginRedirect) this.scheduleLoginRedirectAfterLogout();
        },

        buildItemQueryParams() {
            const params = {};
            const filters = this.itemFilters;
            if (filters.search) params.search = filters.search;
            if (filters.category_id) params.category_ids = filters.category_id;
            if (filters.condition) params.condition = filters.condition;
            if (filters.city_uuid) params.city_uuid = filters.city_uuid;
            if (filters.price_min !== '') params.price_min = filters.price_min;
            if (filters.price_max !== '') params.price_max = filters.price_max;
            if (filters.status) params.status = filters.status;
            if (filters.mine_only && this.me?.id) {
                params.seller_id = this.me.id;
            } else if (filters.seller_id) {
                params.seller_id = filters.seller_id;
            }
            return params;
        },

        itemQueryString(extra = {}, { includePagination = true } = {}) {
            const query = new URLSearchParams();
            const params = includePagination ? { ...this.buildItemQueryParams(), ...extra } : this.buildItemQueryParams();
            for (const [key, value] of Object.entries(params)) {
                if (value !== '' && value != null) query.set(key, value);
            }
            return query.toString();
        },

        itemsRoutePath() {
            const qs = this.itemQueryString({}, { includePagination: false });
            return `/items${qs ? `?${qs}` : ''}`;
        },

        applyItemFiltersFromSearch(search = window.location.search) {
            const params = new URLSearchParams(search);
            const categoryIDs = params.get('category_ids') || params.get('category_id') || '';
            const sellerID = params.get('seller_id') || '';
            this.itemFilters = {
                ...emptyItemFilters(),
                search: params.get('search') || '',
                category_id: categoryIDs.split(',').filter(Boolean)[0] || '',
                condition: params.get('condition') || '',
                city_uuid: params.get('city_uuid') || '',
                price_min: params.get('price_min') || '',
                price_max: params.get('price_max') || '',
                status: params.get('status') || '',
                seller_id: sellerID,
                mine_only: Boolean(sellerID && this.me?.id && sellerID === this.me.id),
            };
        },

        async goToItems({ replace = false, skipRoute = false } = {}) {
            this.mode = MODES.ITEMS;
            if (!skipRoute) this.writeBrowserRoute(this.itemsRoutePath(), { replace });
            await this.loadItems();
        },

        async loadItems() {
            this.itemsLoading = true;
            this.errorMessage = '';
            try {
                const qs = this.itemQueryString({ page: 1, per_page: 20 });
                const data = await this.request(`${this.apiUrl}/items${qs ? `?${qs}` : ''}`, { auth: true });
                this.items = (data.data || []).map((item) => normalizeItem(item, this.locale));
                this.itemsPagination = data.pagination || EMPTY_PAGINATION;
            } catch (error) {
                this.errorMessage = this.getErrorMessage(error, 'Failed to load items');
                this.items = [];
            } finally {
                this.itemsLoading = false;
            }
        },

        applyFilters() {
            this.writeBrowserRoute(this.itemsRoutePath());
            return this.loadItems();
        },

        resetFilters() {
            this.itemFilters = emptyItemFilters();
            this.writeBrowserRoute('/items');
            return this.loadItems();
        },

        async openItemDetail(id, { replace = false, skipRoute = false } = {}) {
            this.mode = MODES.ITEM_DETAIL;
            this.loading = true;
            this.errorMessage = '';
            this.itemDetail = null;
            if (this.promotionPaymentReturnItemID !== id) this.promotionPaymentStatus = '';
            try {
                const data = await this.request(`${this.apiUrl}/items/${id}`, { auth: true });
                this.itemDetail = normalizeItem(data.data || data, this.locale);
                if (!skipRoute) this.writeBrowserRoute(`/items/${id}`, { replace });
            } catch (error) {
                this.errorMessage = this.getErrorMessage(error, 'Failed to load item');
                this.mode = MODES.ITEMS;
                if (!skipRoute) this.writeBrowserRoute('/items', { replace: true });
            } finally {
                this.loading = false;
            }
        },

        goBackFromItemDetail() {
            if (this.itemReturnMode === MODES.FAVORITES) {
                this.goToFavorites();
            } else {
                this.goToItems();
            }
        },

        openItemCreate({ replace = false, skipRoute = false } = {}) {
            this.itemForm = {
                ...emptyItemForm(),
                category_id: this.categories[0]?.id || '',
                city_uuid: this.activeCities()[0]?.id || '',
            };
            this.itemEditId = null;
            this.itemReturnMode = MODES.ITEMS;
            this.itemPhotoIds = [];
            this.itemFormError = '';
            this.mode = MODES.ITEM_CREATE;
            if (!skipRoute) this.writeBrowserRoute('/items/new', { replace });
            this.initItemPhotoPond();
        },

        openItemEdit(item, { replace = false, skipRoute = false } = {}) {
            const normalized = normalizeItem(item, this.locale);
            this.itemEditId = normalized.id;
            this.itemReturnMode = this.mode === MODES.FAVORITES ? MODES.FAVORITES : MODES.ITEMS;
            this.itemPhotoIds = [];
            this.itemFormError = '';
            this.itemForm = {
                title: normalized.title,
                description: normalized.description || '',
                category_id: normalized.category?.id || normalized.category_id || '',
                condition: normalized.condition || 'used',
                city_uuid: normalized.city?.id || normalized.city_uuid || '',
                price_amount: normalized.price?.amount ?? '',
                price_currency: normalized.price?.currency || 'RUB',
                status: normalized.status || 'published',
                tags: (normalized.tags || []).join(', '),
            };
            this.mode = MODES.ITEM_EDIT;
            if (!skipRoute) this.writeBrowserRoute(`/items/${this.itemEditId}/edit`, { replace });
            this.initItemPhotoPond();
        },

        async openItemEditByID(id, { replace = false, skipRoute = false } = {}) {
            this.loading = true;
            this.errorMessage = '';
            try {
                const data = await this.request(`${this.apiUrl}/items/${id}`, { auth: true });
                this.openItemEdit(data.data || data, { replace, skipRoute });
            } catch (error) {
                this.errorMessage = this.getErrorMessage(error, 'Failed to load item');
                await this.goToItems({ replace: true, skipRoute });
            } finally {
                this.loading = false;
            }
        },

        goBackFromItemForm() {
            if (this.itemReturnMode === MODES.FAVORITES) {
                this.goToFavorites();
            } else {
                this.goToItems();
            }
        },

        buildItemPayload({ partial = false } = {}) {
            const form = this.itemForm;
            const payload = {};
            if (!partial || form.title) payload.title = form.title.trim();
            if (!partial || form.description) payload.description = form.description.trim();
            if (!partial || form.category_id) payload.category_id = form.category_id;
            if (!partial || form.condition) payload.condition = form.condition;
            if (!partial || form.city_uuid) payload.city_uuid = form.city_uuid;
            if (form.status) payload.status = form.status;
            if (form.tags) payload.tags = form.tags.split(',').map((tag) => tag.trim()).filter(Boolean);
            if (form.price_amount !== '' && form.price_amount != null) {
                payload.price = {
                    amount: Number(form.price_amount),
                    currency: (form.price_currency || 'RUB').toUpperCase(),
                };
            }
            if (this.itemPhotoIds.length) payload.photo_ids = this.itemPhotoIds;
            return payload;
        },

        validateItemForm() {
            if (!this.itemForm.title || !this.itemForm.category_id || !this.itemForm.city_uuid) {
                this.itemFormError = 'Title, category, and city are required';
                return false;
            }
            return true;
        },

        async handleItemCreate() {
            this.itemFormError = '';
            if (!this.validateItemForm()) return;

            this.itemFormLoading = true;
            try {
                await this.request(`${this.apiUrl}/items`, {
                    method: 'POST',
                    body: this.buildItemPayload(),
                    auth: true,
                });
                await this.goToItems({ replace: true });
            } catch (error) {
                this.itemFormError = this.getErrorMessage(error);
            } finally {
                this.itemFormLoading = false;
            }
        },

        async handleItemUpdate() {
            this.itemFormError = '';
            if (!this.validateItemForm()) return;

            this.itemFormLoading = true;
            try {
                await this.request(`${this.apiUrl}/items/${this.itemEditId}`, {
                    method: 'PATCH',
                    body: this.buildItemPayload({ partial: true }),
                    auth: true,
                });
                if (this.itemReturnMode === MODES.FAVORITES) {
                    await this.goToFavorites({ replace: true });
                } else {
                    await this.goToItems({ replace: true });
                }
            } catch (error) {
                this.itemFormError = this.getErrorMessage(error);
            } finally {
                this.itemFormLoading = false;
            }
        },

        async handleItemDelete(id) {
            if (!confirm('Delete this listing? This cannot be undone.')) return;
            this.loading = true;
            try {
                await this.request(`${this.apiUrl}/items/${id}`, { method: 'DELETE', auth: true, allowStatuses: [204] });
                await this.goToItems({ replace: true });
            } catch (error) {
                this.errorMessage = this.getErrorMessage(error);
            } finally {
                this.loading = false;
            }
        },

        initItemPhotoPond() {
            this.$nextTick(() => {
                const input = document.getElementById('item-photo-input');
                if (!input || !window.FilePond) return;
                if (this.itemPhotoPond) this.itemPhotoPond.destroy();
                this.itemPhotoIds = [];
                this.itemPhotoPond = window.FilePond.create(input, {
                    labelIdle: 'Drag & drop listing photos or <span class="filepond--label-action">Browse</span>',
                    credits: false,
                    allowMultiple: true,
                    maxFiles: 5,
                    server: {
                        process: (fieldName, file, metadata, load, error, progress) => {
                            const formData = new FormData();
                            formData.append('type', 'item');
                            formData.append('file', file, file.name);
                            const xhr = new XMLHttpRequest();
                            xhr.open('POST', `${this.apiUrl}/upload`);
                            xhr.setRequestHeader('Authorization', `Bearer ${this.currentToken}`);
                            xhr.upload.onprogress = (event) => {
                                if (event.lengthComputable) progress(true, event.loaded, event.total);
                            };
                            xhr.onload = () => {
                                let response = {};
                                try {
                                    response = xhr.responseText ? JSON.parse(xhr.responseText) : {};
                                } catch (parseError) {
                                    response = {};
                                }
                                if (xhr.status >= 200 && xhr.status < 300) {
                                    const id = response.data?.id;
                                    if (id) this.itemPhotoIds.push(id);
                                    load(id || 'ok');
                                    return;
                                }
                                error(response.detail || 'Upload failed');
                            };
                            xhr.onerror = () => error('Network error');
                            xhr.send(formData);
                            return { abort: () => xhr.abort() };
                        },
                        revert: (uniqueFileId, load) => {
                            this.itemPhotoIds = this.itemPhotoIds.filter((id) => id !== uniqueFileId);
                            load();
                        },
                    },
                });
            });
        },

        async goToFavorites({ replace = false, skipRoute = false } = {}) {
            this.mode = MODES.FAVORITES;
            if (!skipRoute) this.writeBrowserRoute('/favorites', { replace });
            await this.loadFavorites();
        },

        async loadFavorites() {
            this.favoritesLoading = true;
            this.errorMessage = '';
            try {
                const data = await this.request(`${this.apiUrl}/items/favorites`, { auth: true });
                this.favorites = (data.data || []).map((fav) => ({
                    ...fav,
                    item_id: fav.item_id,
                    item: normalizeItem(fav.item, this.locale),
                }));
                this.favoritesPagination = data.pagination || EMPTY_PAGINATION;
            } catch (error) {
                this.errorMessage = this.getErrorMessage(error, 'Failed to load favorites');
            } finally {
                this.favoritesLoading = false;
            }
        },

        async addToFavorites(itemId) {
            this.favoritesLoading = true;
            try {
                await this.request(`${this.apiUrl}/items/favorites`, {
                    method: 'POST',
                    body: { item_id: itemId },
                    auth: true,
                });
                if (this.itemDetail) this.itemDetail = normalizeItem({ ...this.itemDetail, is_favorited: true }, this.locale);
            } catch (error) {
                this.errorMessage = this.getErrorMessage(error);
            } finally {
                this.favoritesLoading = false;
            }
        },

        async removeFromFavorites(itemId) {
            this.favoritesLoading = true;
            try {
                await this.request(`${this.apiUrl}/items/favorites/${itemId}`, {
                    method: 'DELETE',
                    auth: true,
                    allowStatuses: [204, 404],
                    on404: 'ignore',
                });
                if (this.itemDetail) this.itemDetail = normalizeItem({ ...this.itemDetail, is_favorited: false }, this.locale);
                if (this.mode === MODES.FAVORITES) {
                    this.favorites = this.favorites.filter((fav) => fav.item_id !== itemId);
                    this.favoritesPagination.total = Math.max(0, this.favoritesPagination.total - 1);
                }
            } catch (error) {
                this.errorMessage = this.getErrorMessage(error);
            } finally {
                this.favoritesLoading = false;
            }
        },

        promotionPriceLabel() {
            const amount = this.promotionPaymentAmount;
            if (amount?.value && amount?.currency) return `${amount.value} ${amount.currency}`;
            return 'listing promotion';
        },

        canPromoteItem(item = this.itemDetail) {
            if (!item?.id || !this.isAuthenticated) return false;
            return item.seller?.id === this.me?.id || item.seller?.user_id === this.me?.id || this.hasAdminRole;
        },

        promotionReturnURL(item = this.itemDetail) {
            const url = new URL(window.location.href);
            url.search = '';
            url.hash = '';
            url.searchParams.set('payment_return', 'promote_listing');
            url.searchParams.set('item_id', item.id);
            return url.toString();
        },

        async startPromotionPayment(item = this.itemDetail) {
            if (!item?.id) return;
            this.promotionPaymentLoading = true;
            this.promotionPaymentError = '';
            this.promotionPaymentStatus = '';
            try {
                const response = await this.request(`${this.apiUrl}/payments`, {
                    method: 'POST',
                    auth: true,
                    body: {
                        purpose: 'promote_listing',
                        subject_id: item.id,
                        return_url: this.promotionReturnURL(item),
                    },
                });
                if (response.amount) this.promotionPaymentAmount = response.amount;
                if (!response.confirmation_url) throw new Error('Payment confirmation URL is missing');
                this.promotionPaymentStatus = 'Opening payment page...';
                this.redirectToPayment(response.confirmation_url);
            } catch (error) {
                this.promotionPaymentError = this.getErrorMessage(error, 'Failed to create payment');
            } finally {
                this.promotionPaymentLoading = false;
            }
        },

        redirectToPayment(url) {
            window.location.assign(url);
        },

        goToProfile({ replace = false, skipRoute = false } = {}) {
            this.mode = MODES.PROFILE;
            if (!skipRoute) this.writeBrowserRoute('/profile', { replace });
            this.errorMessage = '';
            this.profileForm = { name: this.me.name || '' };
            this.profilePreferencesForm = this.profilePreferencesFromMe();
            this.profileStatus = '';
            this.profileError = '';
            this.changePasswordForm = { current_password: '', new_password: '' };
            this.changePasswordStatus = '';
            this.changePasswordError = '';
            this.deleteAccountForm = { password: '' };
            this.deleteAccountError = '';
            this.$nextTick(() => this.initProfileAvatarPond());
        },

        profilePreferencesFromMe() {
            const preferences = this.me?.preferences || {};
            return {
                category_id: preferences.category_id || '',
                city_id: preferences.city_id || '',
                condition: preferences.condition || '',
                price_min: preferences.price_min ?? '',
                price_max: preferences.price_max ?? '',
                search: preferences.search || '',
            };
        },

        initProfileAvatarPond() {
            const input = document.getElementById('profile-avatar-input');
            if (!input || !window.FilePond) return;
            if (this.profileAvatarPond) this.profileAvatarPond.destroy();
            this.profileAvatarPond = window.FilePond.create(input, {
                labelIdle: 'Upload new photo',
                credits: false,
                server: {
                    process: (fieldName, file, metadata, load, error, progress) => {
                        const formData = new FormData();
                        formData.append('type', 'avatar');
                        formData.append('file', file, file.name);
                        const xhr = new XMLHttpRequest();
                        xhr.open('POST', `${this.apiUrl}/upload`);
                        xhr.setRequestHeader('Authorization', `Bearer ${this.currentToken}`);
                        xhr.upload.onprogress = (event) => {
                            if (event.lengthComputable) progress(true, event.loaded, event.total);
                        };
                        xhr.onload = () => {
                            let response = {};
                            try {
                                response = xhr.responseText ? JSON.parse(xhr.responseText) : {};
                            } catch (parseError) {
                                response = {};
                            }
                            if (xhr.status >= 200 && xhr.status < 300) {
                                this.me = { ...this.me, avatar_url: response.data?.url };
                                this.avatarUploadStatus = 'Photo updated';
                                this.avatarUploadError = false;
                                load(response.data?.id || 'ok');
                                return;
                            }
                            error(response.detail || 'Upload failed');
                        };
                        xhr.onerror = () => error('Network error');
                        xhr.send(formData);
                        return { abort: () => xhr.abort() };
                    },
                },
            });
        },

        async handleUpdateProfile() {
            if (!this.profileForm.name) {
                this.profileError = 'Name is required';
                return;
            }
            if (
                this.profilePreferencesForm.price_min !== '' &&
                this.profilePreferencesForm.price_max !== '' &&
                Number(this.profilePreferencesForm.price_min) > Number(this.profilePreferencesForm.price_max)
            ) {
                this.profileError = 'Minimum price must be less than maximum price';
                return;
            }
            this.profileLoading = true;
            this.profileError = '';
            this.profileStatus = '';
            try {
                const preferences = this.buildProfilePreferencesPayload();
                const updated = await this.request(`${this.apiUrl}/user/me`, {
                    method: 'PATCH',
                    auth: true,
                    body: {
                        name: this.profileForm.name,
                        ...(Object.keys(preferences).length > 0 && { preferences }),
                    },
                });
                this.me = { ...this.me, ...updated };
                if (Object.keys(preferences).length > 0) {
                    this.me.preferences = { ...(this.me.preferences || {}), ...preferences };
                }
                this.profileStatus = 'Profile updated successfully';
            } catch (error) {
                this.profileError = this.getErrorMessage(error);
            } finally {
                this.profileLoading = false;
            }
        },

        buildProfilePreferencesPayload() {
            const form = this.profilePreferencesForm;
            const payload = {};
            if (form.category_id) payload.category_id = form.category_id;
            if (form.city_id) payload.city_id = form.city_id;
            if (form.condition) payload.condition = form.condition;
            if (form.price_min !== '') payload.price_min = Number(form.price_min);
            if (form.price_max !== '') payload.price_max = Number(form.price_max);
            if (form.search) payload.search = form.search.trim();
            return payload;
        },

        async handleChangePassword() {
            if (!this.changePasswordForm.current_password || !this.changePasswordForm.new_password) {
                this.changePasswordError = 'Please fill in all fields';
                return;
            }
            this.changePasswordLoading = true;
            this.changePasswordError = '';
            this.changePasswordStatus = '';
            try {
                await this.request(`${this.apiUrl}/auth/change-password`, {
                    method: 'POST',
                    auth: true,
                    on401: 'throw',
                    body: this.changePasswordForm,
                });
                this.changePasswordStatus = 'Password changed successfully';
                this.changePasswordForm = { current_password: '', new_password: '' };
            } catch (error) {
                this.changePasswordError = this.getErrorMessage(error);
            } finally {
                this.changePasswordLoading = false;
            }
        },

        async handleDeleteAccount() {
            if (!this.deleteAccountForm.password) {
                this.deleteAccountError = 'Password is required';
                return;
            }
            if (!confirm('Are you sure? This will permanently delete your account.')) return;
            this.deleteAccountLoading = true;
            try {
                await this.request(`${this.apiUrl}/user/me`, {
                    method: 'DELETE',
                    auth: true,
                    on401: 'throw',
                    allowStatuses: [204],
                    body: { password: this.deleteAccountForm.password },
                });
                this.handleLogout();
            } catch (error) {
                this.deleteAccountError = this.getErrorMessage(error);
            } finally {
                this.deleteAccountLoading = false;
            }
        },

        goToContact({ replace = false, skipRoute = false } = {}) {
            this.mode = MODES.CONTACT;
            if (!skipRoute) this.writeBrowserRoute('/contact', { replace });
            this.contactForm = {
                ...emptyContact(),
                name: this.me.name || '',
                email: this.me.email || '',
            };
            this.contactStatus = '';
            this.contactError = '';
        },

        async submitContactMessage() {
            const form = this.contactForm;
            if (!form.name || !form.email || !form.subject || !form.message) {
                this.contactError = 'Please fill in all fields';
                return;
            }
            this.contactLoading = true;
            this.contactError = '';
            this.contactStatus = '';
            try {
                await this.request(`${this.apiUrl}/contact-messages`, {
                    method: 'POST',
                    body: {
                        name: form.name,
                        email: form.email,
                        subject: form.subject,
                        message: form.message,
                    },
                });
                this.contactStatus = 'Message sent';
                this.contactForm.subject = '';
                this.contactForm.message = '';
            } catch (error) {
                this.contactError = this.getErrorMessage(error);
            } finally {
                this.contactLoading = false;
            }
        },

        async openChatWithUser(peerUserID) {
            if (!peerUserID) {
                this.errorMessage = 'Chat is unavailable for this user';
                return;
            }
            this.mode = MODES.CHAT;
            this.writeBrowserRoute('/chat');
            try {
                const conversation = await this.request(`${this.apiUrl}/chat/conversations`, {
                    method: 'POST',
                    auth: true,
                    body: { peer_user_id: peerUserID },
                });
                await this.loadChatConversations();
                await this.selectChatConversation(this.chatConversations.find((item) => item.id === conversation.id) || conversation);
            } catch (error) {
                this.chatConversationsError = this.getErrorMessage(error, 'Failed to open chat');
            }
        },

        async goToChat({ replace = false, skipRoute = false } = {}) {
            this.mode = MODES.CHAT;
            if (!skipRoute) this.writeBrowserRoute('/chat', { replace });
            await this.loadChatConversations();
            if (!this.chatSelectedConversation && this.chatConversations.length) {
                await this.selectChatConversation(this.chatConversations[0]);
            }
        },

        async loadChatConversations() {
            this.chatConversationsLoading = true;
            this.chatConversationsError = '';
            try {
                this.chatConversations = await this.request(`${this.apiUrl}/chat/conversations`, { auth: true });
            } catch (error) {
                this.chatConversationsError = this.getErrorMessage(error, 'Failed to load conversations');
                this.chatConversations = [];
            } finally {
                this.chatConversationsLoading = false;
            }
        },

        async selectChatConversation(conversation) {
            if (!conversation?.id) return;
            this.chatSelectedConversation = conversation;
            await this.loadChatMessages(conversation.id);
            await this.markChatRead(conversation.id);
            this.startChatPolling(conversation.id);
            this.scrollChatToBottomSoon();
        },

        async loadChatMessages(conversationID, { silent = false } = {}) {
            if (!silent) {
                this.chatMessagesLoading = true;
                this.chatMessagesError = '';
            }
            try {
                const messages = await this.request(`${this.apiUrl}/chat/conversations/${conversationID}/messages?limit=50`, { auth: true });
                this.chatMessages = messages.map((message) => ({
                    ...message,
                    is_mine: message.sender_id === this.me?.id || message.is_mine,
                }));
            } catch (error) {
                if (!silent) this.chatMessagesError = this.getErrorMessage(error, 'Failed to load messages');
            } finally {
                if (!silent) this.chatMessagesLoading = false;
            }
        },

        async markChatRead(conversationID) {
            try {
                await this.request(`${this.apiUrl}/chat/conversations/${conversationID}/read`, {
                    method: 'POST',
                    auth: true,
                    allowStatuses: [204],
                });
            } catch (error) {
                // Read markers are non-blocking.
            }
        },

        startChatPolling(conversationID) {
            this.stopChatPolling();
            this.chatRealtimeTimer = window.setInterval(() => {
                if (this.mode === MODES.CHAT && this.chatSelectedConversation?.id === conversationID) {
                    this.loadChatMessages(conversationID, { silent: true });
                }
            }, 3000);
        },

        stopChatPolling() {
            if (this.chatRealtimeTimer) {
                window.clearInterval(this.chatRealtimeTimer);
                this.chatRealtimeTimer = null;
            }
        },

        async sendChatMessage() {
            const body = this.chatMessageBody.trim();
            if (!this.chatSelectedConversation?.id || !body) return;
            this.chatSending = true;
            this.chatMessagesError = '';
            try {
                const response = await this.request(`${this.apiUrl}/chat/conversations/${this.chatSelectedConversation.id}/messages`, {
                    method: 'POST',
                    auth: true,
                    body: { body },
                });
                this.chatMessages = [
                    ...this.chatMessages,
                    { ...response, is_mine: true },
                ];
                this.chatMessageBody = '';
                this.scrollChatToBottomSoon();
            } catch (error) {
                this.chatMessagesError = this.getErrorMessage(error, 'Failed to send message');
            } finally {
                this.chatSending = false;
            }
        },

        chatConversationTitle(conversation) {
            return conversation?.peer_name || conversation?.peer_id || 'User';
        },

        chatMessageTime(message) {
            if (!message?.created_at) return '';
            return new Date(message.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        },

        handleChatInputKeydown(event) {
            if (event.key !== 'Enter' || event.shiftKey) return;
            event.preventDefault();
            this.sendChatMessage();
        },

        scrollChatToBottomSoon() {
            this.$nextTick(() => {
                window.requestAnimationFrame(() => {
                    const box = document.getElementById('chat-messages');
                    if (box) box.scrollTop = box.scrollHeight;
                });
            });
        },

        storageURL(path) {
            return buildStorageURL(this.apiUrl, path);
        },
    };
}
