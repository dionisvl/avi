export function cityName(city, locale = 'en') {
    if (!city) return '';
    if (typeof city === 'string') return city;
    return city.names?.[locale] || city.names?.en || city.names?.ru || city.slug || '';
}

export function activeCities(cities) {
    return Array.isArray(cities) ? cities.filter((city) => city.is_active) : [];
}

export function firstActiveCityID(cities) {
    return activeCities(cities)[0]?.id || '';
}
