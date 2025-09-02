// const API_BASE_URL = 'http://localhost:8081/api/v1';
const API_BASE_URL = 'http://localhost:8081';

function searchOrder() {
    const orderId = document.getElementById('orderInput').value.trim();

    if (!orderId) {
        alert('Please enter order_uid');
        return;
    }

    if (orderId.length !== 20) {
        alert('Order UID must be exactly 20 symbols');
        return;
    }

    window.location.href = `/result.html?order_uid=${encodeURIComponent(orderId)}`;
}

document.addEventListener('DOMContentLoaded', function() {
    const urlParams = new URLSearchParams(window.location.search);
    const orderUid = urlParams.get('order_uid');

    if (orderUid) {
        loadOrderData(orderUid);
    } else {
        showError('Order UID not specified in URL');
    }
});

function loadOrderData(orderUid) {
    showLoading();
    hideError();
    hideResult();

    const startTime = performance.now();

    fetch(`${API_BASE_URL}/order/${orderUid}`)
        .then(response => {
            if (!response.ok) {
                return response.json().then(err => {
                    throw new Error(err.message || 'Server error');
                });
            }
            const clientTime = Math.round(performance.now() - startTime);
            const serverTime = response.headers.get('X-Execution-Time-MS');

            return response.json().then(order => ({
                order,
                execution_time: serverTime ? parseInt(serverTime) : clientTime
            }));
        })
        .then(data => {
            displayOrder(data.order);
            showExecutionTime(data.execution_time);
            hideLoading();
            showResult();
        })
        .catch(error => {
            hideLoading();
            showError(error.message || 'Error occurred while loading the order.');
        });
}

function showExecutionTime(timeMs) {
    const timeElement = document.getElementById('execution-time') || createTimeElement();
    timeElement.textContent = `Request completed in ${timeMs} ms`;
}

function createTimeElement() {
    const timeDiv = document.createElement('div');
    timeDiv.id = 'execution-time';
    timeDiv.className = 'execution-time';
    timeDiv.style.cssText = `
        background: #e8f0fe;
        color: #4285f4;
        padding: 8px 16px;
        border-radius: 4px;
        margin-bottom: 20px;
        font-size: 14px;
        text-align: center;
        font-family: monospace;
        border: 1px solid #4285f4;
    `;

    const resultDiv = document.getElementById('result');
    resultDiv.insertBefore(timeDiv, resultDiv.firstChild);
    return timeDiv;
}

function displayOrder(order) {
    setValue('orderUid', order.order_uid);
    setValue('trackNumber', order.track_number);
    setValue('entry', order.entry);
    setValue('locale', order.locale);
    setValue('internalSignature', order.internal_signature || '');
    setValue('customerId', order.customer_id);
    setValue('deliveryService', order.delivery_service);
    setValue('shardkey', order.shardkey);
    setValue('smId', order.sm_id);
    setValue('dateCreated', formatDate(order.date_created));
    setValue('oofShard', order.oof_shard);

    if (order.delivery) {
        setValue('deliveryName', order.delivery.name);
        setValue('deliveryPhone', order.delivery.phone);
        setValue('deliveryZip', order.delivery.zip);
        setValue('deliveryCity', order.delivery.city);
        setValue('deliveryAddress', order.delivery.address);
        setValue('deliveryRegion', order.delivery.region);
        setValue('deliveryEmail', order.delivery.email);
    }

    if (order.payment) {
        setValue('paymentTransaction', order.payment.transaction);
        setValue('paymentRequestId', order.payment.request_id || '');
        setValue('paymentCurrency', order.payment.currency);
        setValue('paymentProvider', order.payment.provider);
        setValue('paymentAmount', order.payment.amount);
        setValue('paymentDt', formatTimestamp(order.payment.payment_dt));
        setValue('paymentBank', order.payment.bank);
        setValue('paymentDeliveryCost', order.payment.delivery_cost);
        setValue('paymentGoodsTotal', order.payment.goods_total);
        setValue('paymentCustomFee', order.payment.custom_fee);
    }

    displayItems(order.items || []);
}

function displayItems(items) {
    const itemsContainer = document.getElementById('itemsList');
    const itemsCount = document.getElementById('itemsCount');

    itemsCount.textContent = items.length;
    itemsContainer.innerHTML = '';

    if (items.length === 0) {
        itemsContainer.innerHTML = '<p>No items in order</p>';
        return;
    }

    items.forEach((item, index) => {
        const itemElement = document.createElement('div');
        itemElement.className = 'item-card';
        itemElement.innerHTML = `
            <h4>Item ${index + 1}</h4>
            <div class="item-details">
                <div>
                    <span class="info-label">chrt_id:</span>
                    <span class="info-value">${item.chrt_id}</span>
                </div>
                <div>
                    <span class="info-label">track_number:</span>
                    <span class="info-value">${item.track_number}</span>
                </div>
                <div>
                    <span class="info-label">price:</span>
                    <span class="info-value">${item.price}</span>
                </div>
                <div>
                    <span class="info-label">rid:</span>
                    <span class="info-value">${item.rid}</span>
                </div>
                <div>
                    <span class="info-label">name:</span>
                    <span class="info-value">${item.name}</span>
                </div>
                <div>
                    <span class="info-label">sale:</span>
                    <span class="info-value">${item.sale}%</span>
                </div>
                <div>
                    <span class="info-label">size:</span>
                    <span class="info-value">${item.size}</span>
                </div>
                <div>
                    <span class="info-label">total_price:</span>
                    <span class="info-value">${item.total_price}</span>
                </div>
                <div>
                    <span class="info-label">nm_id:</span>
                    <span class="info-value">${item.nm_id}</span>
                </div>
                <div>
                    <span class="info-label">brand:</span>
                    <span class="info-value">${item.brand}</span>
                </div>
                <div>
                    <span class="info-label">status:</span>
                    <span class="info-value">${getStatusText(item.status)}</span>
                </div>
            </div>
        `;
        itemsContainer.appendChild(itemElement);
    });
}

function setValue(elementId, value) {
    const element = document.getElementById(elementId);
    if (element) {
        element.textContent = value !== null && value !== undefined ? value : '';
    }
}

function formatDate(dateString) {
    if (!dateString) return '';
    const date = new Date(dateString);
    return date.toLocaleString('ru-RU');
}

function formatTimestamp(timestamp) {
    if (!timestamp) return '';
    const date = new Date(timestamp * 1000);
    return date.toLocaleString('ru-RU');
}

function getStatusText(statusCode) {
    const statusMap = {
        202: 'Approved',
        200: 'Created',
        300: 'Sale',
        400: 'Not available'
    };
    return statusMap[statusCode] || statusCode;
}

function showLoading() {
    document.getElementById('loading').classList.remove('hidden');
}

function hideLoading() {
    document.getElementById('loading').classList.add('hidden');
}

function showError(message) {
    const errorElement = document.getElementById('error');
    const errorMessage = document.getElementById('errorMessage');

    errorMessage.textContent = message;
    errorElement.classList.remove('hidden');
}

function hideError() {
    document.getElementById('error').classList.add('hidden');
}

function showResult() {
    document.getElementById('result').classList.remove('hidden');
}

function hideResult() {
    document.getElementById('result').classList.add('hidden');
}

if (document.getElementById('orderInput')) {
    document.getElementById('orderInput').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            searchOrder();
        }
    });

    document.getElementById('orderInput').focus();
}